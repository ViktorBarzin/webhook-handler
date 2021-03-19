package chatbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"viktorbarzin/webhook-handler/chatbot/auth"
	"viktorbarzin/webhook-handler/chatbot/executor"
	"viktorbarzin/webhook-handler/chatbot/fbapi"
	"viktorbarzin/webhook-handler/chatbot/models"
	"viktorbarzin/webhook-handler/chatbot/statemachine"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type MessageType int
type MoveFSMResult struct {
	Ok            bool
	CmdOutput     string
	PrintStateMsg bool
	// String to append to CmdOutput
	AdditionalMsg string
}

const (
	Raw MessageType = iota
	Postback
)

// ChatbotHandler is a HTTP handler which keeps track of conversations
type ChatbotHandler struct {
	UserToFSM   map[string]*statemachine.FSMWithStatesAndEvents
	ConfigFile  string
	States      []statemachine.State
	Events      []statemachine.Event
	RBACConfig  auth.RBACConfig
	fsmTemplate statemachine.FSMWithStatesAndEvents
}

func NewChatbotHandler(configFile string) (*ChatbotHandler, error) {
	rbac, err := auth.NewRBACConfig(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse config file and create RBAC struct")
	}
	f, err := statemachine.ChatBotFSM(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create chatbot FSM ")
	}
	c := &ChatbotHandler{
		UserToFSM:   map[string]*statemachine.FSMWithStatesAndEvents{},
		ConfigFile:  configFile,
		RBACConfig:  rbac,
		fsmTemplate: *f,
	}
	fbapi.SetGetStartedButton()
	return c, nil
}

func (c *ChatbotHandler) HandleFunc(w http.ResponseWriter, r *http.Request) {
	if fbapi.IsVerifyRequest(w, r) {
		glog.Info("verify request processed")
		return
	}

	ok, reason := fbapi.ValidSignature(r)
	if !ok {
		errorMsg := fmt.Sprintf("failed to verify signatures: %s", reason)
		glog.Warning(errorMsg)
		fbapi.ResponseWrite(w, 403, errorMsg)
		return
	}
	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf(fmt.Sprintf("failed to read body for request %+v", r))
		return
	}
	glog.Infof("Processing request body: '%+v'", string(bodybytes))

	messageType, err := getMessageType(string(bodybytes))
	if err != nil {
		err = errors.Wrap(err, "failed to get message type")
		glog.Errorf(fmt.Sprintf("failed processing request: %s", err.Error()))
		fbapi.ResponseWrite(w, 400, "Unsupported message type")
		return
	}

	if messageType == Raw {
		var fbCallbackMsg models.FbMessageCallback
		json.Unmarshal(bodybytes, &fbCallbackMsg)
		glog.Infof("processing raw message: %s", string(bodybytes))
		err = c.processRawMessage(fbCallbackMsg)
	} else if messageType == Postback {
		var fbCallbackMsg models.FbMessagePostBackCallback
		json.Unmarshal(bodybytes, &fbCallbackMsg)
		glog.Infof("processing postback message: %s", string(bodybytes))
		err = c.processPostBackMessage(fbCallbackMsg)
	} else {
		errMsg := "received an unsupported message type"
		glog.Warning(errMsg)
		fbapi.ResponseWrite(w, 400, errMsg)
		return
	}
	if err != nil {
		glog.Errorf(fmt.Sprintf("failed processing message: %s", err.Error()))
		fbapi.ResponseWrite(w, 500, "Internal error")
	} else {
		glog.Infof("Successfully processed request with body: '%s'", string(bodybytes))
		fbapi.ResponseWrite(w, 200, "OK")
	}
}

func (c *ChatbotHandler) processRawMessage(fbCallbackMsg models.FbMessageCallback) error {
	for _, e := range fbCallbackMsg.Entry {
		for _, m := range e.Messaging {
			if err := c.processMessage(m.Sender.ID, m.Message.Text); err != nil {
				return errors.Wrapf(err, "failed processing raw message")
			}
		}
	}
	return nil
}

func (c *ChatbotHandler) processPostBackMessage(fbCallbackMsg models.FbMessagePostBackCallback) error {
	for _, e := range fbCallbackMsg.Entry {
		for _, m := range e.Messaging {
			if err := c.processMessage(m.Sender.ID, m.Postback.Payload); err != nil {
				return errors.Wrapf(err, "failed processing postback message")
			}
		}
	}
	return nil
}

func (c *ChatbotHandler) processMessage(senderID, payload string) error {
	userFsm, ok := c.UserToFSM[senderID]
	if !ok {
		c.UserToFSM[senderID] = &c.fsmTemplate
		userFsm = c.UserToFSM[senderID]
	}

	// Try make transition
	oldState := userFsm.Current()
	user := c.RBACConfig.WhoAmI(senderID)
	glog.Infof("attempting to send %s for user %+v", payload, user)
	moveFSMResult, err := c.moveFSM(user, userFsm, payload)

	if err == nil {
		glog.Infof("successful transition from '%s' with msg: '%s' to '%s'. Available transitions are: %+v", oldState.Name, payload, userFsm.Current().Name, userFsm.FSM.AvailableTransitions())
		// Execute command at current state if allowed
		if c.RBACConfig.IsAllowedToExecuteBatch(user, userFsm.Current().Commands) {
			glog.Infof("user %+v is allowed to execute commands: %+v", user, userFsm.Current().Commands)
			for _, cmd := range userFsm.Current().Commands {
				executor.Execute(cmd)
			}
		} else {
			glog.Infof("user %+v is not allowed to execute 1 or more of the commands: %+v", user, userFsm.Current().Commands)
		}
	} else {
		glog.Warningf("failed to make transition from '%s' with msg '%s'. Available transitions are: %+v", oldState.Name, payload, userFsm.FSM.AvailableTransitions())
	}

	respondToUser(senderID, *userFsm, moveFSMResult)
	return nil
}

func (c *ChatbotHandler) resetFSM(userFsm map[string]*statemachine.FSMWithStatesAndEvents, userid string) {
	f, _ := statemachine.ChatBotFSM(c.ConfigFile)
	userFsm[userid] = f
}

func respondToUser(recipient string, userFsm statemachine.FSMWithStatesAndEvents, moveFSMResult MoveFSMResult) error {
	// Send raw message with long explanation
	msgToSend := ""
	if moveFSMResult.CmdOutput != "" {
		// if cmd output is non-empty send that
		msgToSend = moveFSMResult.CmdOutput
		if moveFSMResult.AdditionalMsg != "" {
			// If appendix is non-empty, append it ot msg
			msgToSend += moveFSMResult.AdditionalMsg
		}
	} else {
		currentState := userFsm.Current()
		msgToSend = currentState.Message
	}
	fbapi.SendRawMessage(recipient, msgToSend)

	// Create postback with options to choose from next
	events := userFsm.AvailableTransitions()
	events = statemachine.Sorted(events)
	buttons := eventsToPostbackButtons(events)
	elements := getPostbackElements("What's next?", "Tap to answer", buttons)
	// Get consistent button order
	payload := getPostbackPaylod(recipient, elements)
	fbapi.SendPostBackMessage(recipient, payload)
	return nil
}

// Given a user state machine and a message, try to make a transition and create a response
func (h ChatbotHandler) moveFSM(user auth.User, userFsm *statemachine.FSMWithStatesAndEvents, event string) (MoveFSMResult, error) {
	// If transition is allowed in state machine
	if userFsm.FSM.Can(event) {
		// move to state and check permission. if not allowed, revert
		oldState := userFsm.FSM.Current()
		err := userFsm.FSM.Event(event)
		if err != nil {
			return MoveFSMResult{}, errors.Wrapf(err, "failed to move")
		}
		// If user is not allowed to be in new state, revert
		if !h.RBACConfig.IsAllowedMany(user, auth.ToPermissions(userFsm.Current().Permissions)) {
			userFsm.FSM.SetState(oldState)
			return MoveFSMResult{}, fmt.Errorf("user %+v does not have permission for state %+v. returning to %s", user, userFsm.FSM.Current(), oldState)
		}
		return MoveFSMResult{}, nil
	} else {
		// Uncomment below to enable special state handling
		// if callback, ok := statemachine.SpecialStateTypeCallback[userFsm.Current().SpecialStateType]; ok {
		// 	glog.Infof("executing special state handler for state %+v, event: %s", userFsm.Current(), event)
		// 	output, err := callback(event)
		// 	if err != nil {
		// 		return MoveFSMResult{Ok: false, PrintStateMsg: false}, errors.Wrapf(err, "failed to execute special callback")
		// 	}
		// 	additionalMsg := "Successfully added your config!\n\nThe last thing you need to do is update the [Interface] IP address in your config. Please set it to match the one in the command output above.\n\nOnce you do that you are ready to go! Please wait for a couple of minutes before using your new config so the backend system changes can propagate."
		// 	return MoveFSMResult{Ok: true, CmdOutput: output, PrintStateMsg: false, AdditionalMsg: additionalMsg}, nil
		// }
		return MoveFSMResult{}, fmt.Errorf("cannot process %s", event)
	}
}

func getPostbackElements(title, subtitle string, buttons []models.MessageWithPostbackButton) []models.MessageWithPostbackElement {
	// Fb allows only 3 buttons per element, so group elements
	elements := []models.MessageWithPostbackElement{}
	buttonGroup := []models.MessageWithPostbackButton{}
	for i, b := range buttons {
		if i > 0 && i%3 == 0 {
			elements = append(elements,
				models.MessageWithPostbackElement{
					Title:    title,
					Subtitle: subtitle,
					Buttons:  buttonGroup,
				},
			)
			buttonGroup = []models.MessageWithPostbackButton{}
		}
		buttonGroup = append(buttonGroup, b)
	}
	if len(buttonGroup) > 0 {
		elements = append(elements,
			models.MessageWithPostbackElement{
				Title:    title,
				Subtitle: subtitle,
				Buttons:  buttonGroup,
			},
		)
	}
	return elements
}

func getPostbackPaylod(recipient string, elements []models.MessageWithPostbackElement) models.PayloadPostback {
	return models.PayloadPostback{
		Recipient: models.Recipient{
			ID: recipient,
		},
		Message: models.MessageWithPostback{
			Attachment: models.MessageWithPostbackAttachment{
				Type: "template",
				Payload: models.MessageWithPostbackPayload{
					TemplateType: "generic",
					Elements:     elements,
				},
			},
		},
	}
}

func eventsToPostbackButtons(events []statemachine.Event) []models.MessageWithPostbackButton {
	res := []models.MessageWithPostbackButton{}
	for _, e := range events {
		b := models.MessageWithPostbackButton{
			Type:    "postback",
			Title:   e.Message,
			Payload: e.Name,
		}
		res = append(res, b)
	}
	return res
}

func getMessageType(jsonBody string) (MessageType, error) {
	var rawMsg map[string]interface{}
	err := json.Unmarshal([]byte(jsonBody), &rawMsg)

	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("failed to json decode string: '%s'", jsonBody))
	}

	// extract "entry" list
	rawEntries, ok := rawMsg["entry"]
	if !ok {
		return 0, errors.New(fmt.Sprintf("'entry' key not found for msg: %s", jsonBody))
	}
	entries, ok := rawEntries.([]interface{})
	if !ok {
		return 0, errors.New(fmt.Sprintf("'entry' key is not a list"))
	}

	entry, ok := entries[0].(map[string]interface{})
	if !ok {
		return 0, errors.New(fmt.Sprintf("'entry' list is empty. 1 element of type map[string]interface{} required"))
	}

	// if "messaging" key is present it is a raw message
	rawMessaging, ok := entry["messaging"]
	if !ok {
		return 0, errors.New(fmt.Sprintf("'messaging' key not found for msg: %s", jsonBody))
	}
	messaging, ok := rawMessaging.([]interface{})
	if !ok {
		return 0, errors.New(fmt.Sprintf("'messaging' key is not a list"))
	}

	message, ok := messaging[0].(map[string]interface{})
	if !ok {
		return 0, errors.New(fmt.Sprintf("'messaging' list is empty. 1 element of type map[string]interface{} required"))
	}

	// Check type
	if _, ok := message["message"]; ok {
		return Raw, nil
	}
	if _, ok := message["postback"]; ok {
		return Postback, nil
	}
	return 0, errors.New(fmt.Sprintf("message type is not supported. message: %s", jsonBody))
}
