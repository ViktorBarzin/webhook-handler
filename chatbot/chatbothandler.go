package chatbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"

	"github.com/viktorbarzin/webhook-handler/chatbot/auth"
	"github.com/viktorbarzin/webhook-handler/chatbot/executor"
	"github.com/viktorbarzin/webhook-handler/chatbot/fbapi"
	"github.com/viktorbarzin/webhook-handler/chatbot/models"
	"github.com/viktorbarzin/webhook-handler/chatbot/statemachine"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type MessageType int
type MoveFSMResult struct {
	CmdOutput     string
	AdditionalMsg string
	FSM           statemachine.FSMWithStatesAndEvents
}

const (
	Raw MessageType = iota
	Postback
)

var (
	allowedUserInputRe = regexp.MustCompile(`^[a-zA-Z0-9=.@ ]{1,500}$`)

	// vpnFriendlyNameRegex = regexp.MustCompile(`(\w| ){1,40}`)
	// vpnPubKeyRegex       = regexp.MustCompile(`[-A-Za-z0-9+=]{1,50}|=[^=]|={3,}`)
)

// ChatbotHandler is a HTTP handler which keeps track of conversations
type ChatbotHandler struct {
	UserToFSM                 map[string]*statemachine.FSMWithStatesAndEvents
	ConfigFile                string
	States                    []statemachine.State
	Events                    []statemachine.Event
	RBACConfig                auth.RBACConfig
	ProcessedApprovalRequests []string
}

func NewChatbotHandler(configFile string) (*ChatbotHandler, error) {
	rbac, err := auth.NewRBACConfig(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse config file and create RBAC struct")
	}
	_, err = statemachine.ChatBotFSM(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create chatbot FSM from config file %s", configFile)
	}
	c := &ChatbotHandler{
		UserToFSM:  map[string]*statemachine.FSMWithStatesAndEvents{},
		ConfigFile: configFile,
		RBACConfig: rbac,
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
		fbapi.ResponseWrite(w, 200, "Internal error")
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
		f, err := statemachine.ChatBotFSM(c.ConfigFile)
		if err != nil {
			return errors.Wrapf(err, "failed to create chatbot FSM for user id %s", senderID)
		}
		c.UserToFSM[senderID] = f
		userFsm = c.UserToFSM[senderID]
	}
	user := c.RBACConfig.WhoAmI(senderID)
	moveFSMResult := MoveFSMResult{}
	moveFSMResult.FSM = *userFsm

	if isApprovalRequest([]byte(payload)) {
		glog.Infof("Processing approval request: %s", payload)
		return c.processApprovalRequestMessage(senderID, payload, moveFSMResult)
	}
	// Try make transition
	err := c.moveFSM(user, userFsm, payload)

	if err == nil {
		glog.Infof("successful transition from '%s' with msg: '%s' to '%s'. Available transitions are: %+v", userFsm.Current().Name, payload, userFsm.Current().Name, userFsm.FSM.AvailableTransitions())
		// Execute command at current state if allowed
		// if c.RBACConfig.IsAllowedToExecuteMany(user, userFsm.Current().Commands) {
		// 	glog.Infof("user %+v is allowed to execute commands: %+v", user, userFsm.Current().Commands)
		// 	for _, cmd := range userFsm.Current().Commands {
		// 		executor.Execute(cmd)
		// 	}
		// } else {
		// 	glog.Infof("user %+v is not allowed to execute 1 or more of the commands: %+v", user, userFsm.Current().Commands)
		// }
	} else {
		glog.Infof("trying to find a default handler to process '%s'", payload)
		// This is shit, I know, will refactor

		// If default handler is defined
		if !reflect.DeepEqual(userFsm.Current().DefaultHandler, auth.Command{}) {
			glog.Infof("found default handler")
			// if user is allowed to execute default handler
			if c.RBACConfig.IsAllowedToExecute(user, userFsm.Current().DefaultHandler) {
				glog.Infof("executing default handler '%s' for user '%s' state '%s'", userFsm.Current().DefaultHandler.PrettyName, user.Name, userFsm.Current().Name)
				// if input to handler is not valid
				if !isValidUserInput(payload) {
					glog.Warningf("user input '%s' did not match allowed pattern '%s'", payload, allowedUserInputRe.String())
					moveFSMResult.AdditionalMsg = fmt.Sprintf("Invalid input. Please stick to using charates from '%s' set.", allowedUserInputRe.String())
				} else {
					glog.Infof("user input '%s' allowed, proceeding with executing default handler", payload)
					moveFSMResult.CmdOutput = "Your command has been scheduled for execution, stand by for results..."
					go executeAndRepond(senderID, moveFSMResult, userFsm.Current().DefaultHandler, payload)
				}
			} else {
				// if not allowed to execute default handler
				glog.Warningf("found default handler '%s' to execute but user '%s' does not have permission to execute this command", userFsm.Current().DefaultHandler.PrettyName, user.Name)
				glog.Infof("sending approval request")
				if err := c.sendRequestApprovalRequest(user, userFsm.Current().DefaultHandler, payload); err != nil {
					moveFSMResult.CmdOutput = fmt.Sprintf("failed to send permission approval request : %s", err.Error())
				} else {
					moveFSMResult.CmdOutput = fmt.Sprintf("You do not have permission to execute '%s'. I have asked for a review for your request. Please standby...", userFsm.Current().DefaultHandler.PrettyName)
				}
			}
		} else {
			// not a valid event, no defined handler
			glog.Warningf("failed to make transition from '%s' with msg '%s'. Available transitions are: %+v", userFsm.Current().Name, payload, userFsm.FSM.AvailableTransitions())
			moveFSMResult.AdditionalMsg = "Could not understand your message :/ Please stick to using the button unless otherwise asked :-)"
		}
	}

	return respondToUser(senderID, moveFSMResult)
}

func (c *ChatbotHandler) processApprovalRequestMessage(senderID, payload string, moveFSMResult MoveFSMResult) error {
	user := c.RBACConfig.WhoAmI(senderID)

	req, _ := DeserializeApprovalRequest([]byte(payload))
	if c.isProcessed(req.ID) {
		cmd, _ := c.cmdFromId(req.CmdID)
		fbapi.SendRawMessage(senderID, fmt.Sprintf("approval request for '%s' has already been processed", cmd.PrettyName))
		return nil
	}
	what, err := c.cmdFromId(req.CmdID)
	if err != nil {
		fbapi.SendRawMessage(senderID, fmt.Sprintf("failed to find command with id '%s'", req.CmdID))
		return err
	}
	// if sender is authorized to process this request
	if c.RBACConfig.UserHasRole(user, what.ApprovedBy) {
		// user authorized
		c.SendApprovalRequestUpdateNotification(req, user)
		if req.State == ApprovalStateAccepted {
			fbapi.SendRawMessage(req.From.ID, fmt.Sprintf("Command '%s' with input '%s' will begin executing shortly...", what.PrettyName, req.Payload))

			go executeAndRepond(req.From.ID, moveFSMResult, what, req.Payload)
			// output, err := executor.Execute(what, req.Payload)
			// moveFSMResult.CmdOutput = output
			// if err != nil {
			// 	moveFSMResult.AdditionalMsg = fmt.Sprintf("command '%s' failed: %s", what.PrettyName, err.Error())
			// } else {
			// 	moveFSMResult.AdditionalMsg = "Success!"
			// }
		} else if req.State == ApprovalStateRejected {

		} else {
			// unknown approval state request
		}
	} else {
		// user not allowed to authrozie this request
		moveFSMResult.AdditionalMsg = fmt.Sprintf("Permission denied. You do not have '%s' role", what.ApprovedBy.Name)
	}
	if err := respondToUser(req.From.ID, moveFSMResult); err != nil {
		return errors.Wrapf(err, "failed to notify request creator about the outcome of their command")
	}

	if err := respondToUser(senderID, moveFSMResult); err != nil {
		return errors.Wrapf(err, "failed to notify request moderator about the outcome of the command")
	}
	return nil
}

func (c *ChatbotHandler) resetFSM(userFsm map[string]*statemachine.FSMWithStatesAndEvents, userid string) {
	f, _ := statemachine.ChatBotFSM(c.ConfigFile)
	userFsm[userid] = f
}

func (c *ChatbotHandler) isProcessed(requestGuid string) bool {
	for _, p := range c.ProcessedApprovalRequests {
		if p == requestGuid {
			return true
		}
	}
	return false
}

func isValidUserInput(input string) bool {
	return allowedUserInputRe.Match([]byte(input))
}

func respondToUser(recipient string, moveFSMResult MoveFSMResult) error {
	// Send raw message with long explanation
	msgToSend := ""
	msgToSend += moveFSMResult.CmdOutput
	// if moveFSMResult.CmdOutput != "" {
	// if cmd output is non-empty send that
	// msgToSend += moveFSMResult.CmdOutput
	// msgToSend += fmt.Sprintf("Command output: ```\n\n%s\n\n```", moveFSMResult.CmdOutput)
	// }
	if moveFSMResult.AdditionalMsg != "" {
		// If appendix is non-empty, append it to msg
		msgToSend += moveFSMResult.AdditionalMsg
	}
	if moveFSMResult.CmdOutput == "" {
		if msgToSend != "" {
			msgToSend += "\n\n"
		}
		msgToSend += moveFSMResult.FSM.Current().Message
	}
	fbapi.SendRawMessage(recipient, msgToSend)

	// Create postback with options to choose from next
	events := statemachine.Sorted(moveFSMResult.FSM.AvailableTransitions())
	buttons := eventsToPostbackButtons(events)
	elements := getPostbackElements("What's next?", "Tap to answer", buttons)
	// Get consistent button order
	payload := getPostbackPayload(recipient, elements)
	fbapi.SendPostBackMessage(recipient, payload)
	return nil
}

// Given a user state machine and a message, try to make a transition and create a response
func (h ChatbotHandler) moveFSM(user auth.User, userFsm *statemachine.FSMWithStatesAndEvents, event string) error {
	// If transition is allowed in state machine
	if userFsm.FSM.Can(event) {
		// move to state and check permission. if not allowed, revert
		oldState := userFsm.FSM.Current()
		err := userFsm.FSM.Event(event)
		if err != nil {
			return errors.Wrapf(err, "failed to move")
		}
		// If user is not allowed to be in new state, revert
		if !h.RBACConfig.IsAllowedMany(user, auth.ToPermissions(userFsm.Current().Permissions)) {
			userFsm.FSM.SetState(oldState)
			return fmt.Errorf("user %+v does not have permission for state %+v. returning to %s", user, userFsm.FSM.Current(), oldState)
		}
		return nil
	}
	// else {
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
	return fmt.Errorf("cannot process %s", event)
	// }
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

func getPostbackPayload(recipient string, elements []models.MessageWithPostbackElement) models.PayloadPostback {
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

func executeAndRepond(senderID string, moveFSMResult MoveFSMResult, cmd auth.Command, payload string) {
	cmdOutput, err := executor.Execute(cmd, payload)
	// if no error during execution of handler
	if err == nil {
		glog.Infof("successfully executed default handler for input '%s'", payload)
		if cmd.ShowCmdOutput {
			moveFSMResult.CmdOutput = cmdOutput
		}
		moveFSMResult.AdditionalMsg = cmd.SuccessExplanation
	} else {
		// if error during execution of handler
		errMsg := fmt.Sprintf("failed to execute handler func: %s", err.Error())
		glog.Errorf(errMsg)
		moveFSMResult.AdditionalMsg = errMsg
	}
	respondToUser(senderID, moveFSMResult)
}
