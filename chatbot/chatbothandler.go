package chatbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"sort"
	"viktorbarzin/webhook-handler/chatbot/models"
	"viktorbarzin/webhook-handler/chatbot/statemachine"

	"github.com/golang/glog"
	"github.com/looplab/fsm"
	"github.com/pkg/errors"
)

type MessageType int

const (
	Raw MessageType = iota
	Postback
)

// ChatbotHandler is a HTTP handler which keeps track of conversations
type ChatbotHandler struct {
	UserToFSM map[string]*fsm.FSM
}

func NewChatbotHandler() *ChatbotHandler {
	return &ChatbotHandler{UserToFSM: map[string]*fsm.FSM{}}
}

func (c *ChatbotHandler) HandleFunc(w http.ResponseWriter, r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, true)
	glog.Infof("Processing: '%+v'", string(requestDump))
	if verifyRequest(w, r) {
		glog.Info("verify request processed")
		return
	}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "error reading body")
	}

	messageType, err := getMessageType(string(bodybytes))
	if err != nil {
		err = errors.Wrap(err, "failed to get message type")
		glog.Errorf(fmt.Sprintf("failed processing request: %s", err.Error()))
		return
	}

	if messageType == Raw {
		var fbCallbackMsg models.FbMessageCallback
		json.Unmarshal(bodybytes, &fbCallbackMsg)
		err = c.processRawMessage(fbCallbackMsg)
	} else if messageType == Postback {
		var fbCallbackMsg models.FbMessagePostBackCallback
		json.Unmarshal(bodybytes, &fbCallbackMsg)
		err = c.processPostBackMessage(fbCallbackMsg)
	} else {
		err = errors.New("received an unsupported message type")
	}
	if err != nil {
		glog.Errorf(fmt.Sprintf("failed processing message: %s", err.Error()))
	}
}

func (c *ChatbotHandler) processRawMessage(fbCallbackMsg models.FbMessageCallback) error {
	for _, e := range fbCallbackMsg.Entry {
		for _, m := range e.Messaging {
			userFsm, ok := c.UserToFSM[m.Sender.ID]
			if !ok {
				c.UserToFSM[m.Sender.ID] = statemachine.ChatBotFSM()
				userFsm = c.UserToFSM[m.Sender.ID]
			}

			msg := "I am not smart enough to understand any message :/ Please stick to using the buttons I know how to answer :)"
			if userFsm.Current() == statemachine.InitialStateName {
				msg = "Thank you reaching out to me. Let's get you started!"
			}
			rawMsg := getRawMsg(m.Sender.ID, msg)
			reader, err := rawMsgPayloadReader(rawMsg, m.Sender.ID)
			if err != nil {
				glog.Warningf("failed to create message reader for message: %+v", rawMsg)
				continue
			}
			sendRequest(reader)

			// Send the "Get Started" postback
			postbackMsg := getPostbackMsg(userFsm, statemachine.Help.Name, m.Sender.ID)
			reader, err = postbackPayloadReader(postbackMsg, m.Sender.ID)
			if err != nil {
				glog.Warningf("failed to create message reader for message: %+v", postbackMsg)
				continue
			}
			sendRequest(reader)
		}
	}
	return nil
}

func (c *ChatbotHandler) processPostBackMessage(fbCallbackMsg models.FbMessagePostBackCallback) error {
	for _, e := range fbCallbackMsg.Entry {
		for _, m := range e.Messaging {
			userFsm, ok := c.UserToFSM[m.Sender.ID]
			if !ok {
				c.UserToFSM[m.Sender.ID] = statemachine.ChatBotFSM()
				userFsm = c.UserToFSM[m.Sender.ID]
			}
			postbackMsg := getPostbackMsg(userFsm, m.Postback.Payload, m.Sender.ID)
			msg, err := postbackPayloadReader(postbackMsg, m.Sender.ID)
			if err != nil {
				glog.Warningf("failed to create message reader for message: %+v", postbackMsg)
				continue
			}
			sendRequest(msg)
		}
	}
	return nil
}

func verifyRequest(w http.ResponseWriter, r *http.Request) bool {
	urlVals := r.URL.Query()
	mode := urlVals.Get("hub.mode")
	token := urlVals.Get("hub.verify_token")
	challenge := urlVals.Get("hub.challenge")
	if mode != "" && token != "" {
		if mode == "subscribe" && token == verifyToken {
			glog.Info("webhook verified")
			w.WriteHeader(200)
			w.Write([]byte(challenge))
		} else {
			w.WriteHeader(403)
		}
		return true
	}
	return false
}

func getRawMsg(receiver, msg string) models.Payload {
	return models.Payload{
		Recipient: models.Recipient{ID: receiver},
		Message: models.Message{
			Text: msg,
		},
	}
}

// Given a user state machine and a message, try to make a transition and create a response
func getPostbackMsg(userFsm *fsm.FSM, msg string, recipient string) models.PayloadPostback {
	subtitle := "Tap to answer"

	var title string
	// If transition is allowed
	if userFsm.Can(msg) {
		current := userFsm.Current()
		userFsm.Event(msg)
		newState := statemachine.StateFromString(userFsm.Current())
		title = newState.Message
		glog.Infof("successful transition from '%s' with msg: '%s' to '%s'. Available transitions are: %+v", current, msg, userFsm.Current(), userFsm.AvailableTransitions())
	} else {
		title = "Oops, I didn't quite get that, please try again\n\n" + statemachine.StateFromString(userFsm.Current()).Message
		glog.Warningf("failed transition from '%ss' with msg: %s", userFsm.Current(), msg)
	}
	transitions := userFsm.AvailableTransitions()
	// Get consistent button order
	sort.Strings(transitions)
	buttons := transitionsToPostbackButtons(transitions)

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
			// buttonGroup
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

func transitionsToPostbackButtons(transitions []string) []models.MessageWithPostbackButton {
	res := []models.MessageWithPostbackButton{}
	for _, t := range transitions {
		event := statemachine.EventFromString(t)
		b := models.MessageWithPostbackButton{
			Type:    "postback",
			Title:   event.Message,
			Payload: event.Name,
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
