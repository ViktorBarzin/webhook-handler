package chatbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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
	c := &ChatbotHandler{UserToFSM: map[string]*fsm.FSM{}}
	c.setGetStartedButton()
	return c
}

func (c *ChatbotHandler) setGetStartedButton() error {
	getStartedButtonPayload := map[string]map[string]string{
		"get_started": {"payload": statemachine.GetStartedEventName},
	}
	marshalled, err := json.Marshal(getStartedButtonPayload)
	if err != nil {
		return errors.Wrap(err, "failed to marshall get started button payload")
	}
	reader := bytes.NewReader(marshalled)
	resp, err := sendRequestURI("https://graph.facebook.com/v2.6/me/messenger_profile", reader)
	if err != nil {
		return errors.Wrap(err, "failed sending request")
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed reading response body")
	}
	glog.Infof("Received response to setting payload button: '%s'", respBody)
	return nil
}

func (c *ChatbotHandler) HandleFunc(w http.ResponseWriter, r *http.Request) {
	if isVerifyRequest(w, r) {
		glog.Info("verify request processed")
		return
	}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "error reading body")
	}
	glog.Infof("Processing request body: '%+v'", string(bodybytes))

	messageType, err := getMessageType(string(bodybytes))
	if err != nil {
		err = errors.Wrap(err, "failed to get message type")
		glog.Errorf(fmt.Sprintf("failed processing request: %s", err.Error()))
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
		err = errors.New("received an unsupported message type")
	}
	if err != nil {
		glog.Errorf(fmt.Sprintf("failed processing message: %s", err.Error()))
	} else {
		glog.Infof("Successfully processed request with body: '%s'", string(bodybytes))
	}
}

func (c *ChatbotHandler) processRawMessage(fbCallbackMsg models.FbMessageCallback) error {
	for _, e := range fbCallbackMsg.Entry {
		for _, m := range e.Messaging {
			// If user is not asking for help, tell that we only understand help raw msg
			if strings.ToLower(m.Message.Text) != strings.ToLower(statemachine.HelpEventName) {
				msg := "I am not smart enough to understand that message just yet :/ Please stick to using the buttons I know how to answer :)"
				resp, err := sendRawMessage(m.Sender.ID, msg)
				if err != nil {
					return errors.Wrap(err, "failed to send raw message")
				}
				if resp.StatusCode != http.StatusOK {
					body, _ := ioutil.ReadAll(resp.Body)
					return errors.New(fmt.Sprintf("sending raw message returned a non-OK status code: %d, response body: %s", resp.StatusCode, body))
				}
			}

			// Move FSM with "help"
			userFsm, ok := c.UserToFSM[m.Sender.ID]
			if !ok {
				c.UserToFSM[m.Sender.ID] = statemachine.ChatBotFSM()
				userFsm = c.UserToFSM[m.Sender.ID]
			}
			moveFSM(userFsm, statemachine.HelpEventName)
			respondToUser(*userFsm, m.Sender.ID)
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

			// Try make transition
			oldState := statemachine.StateFromString(userFsm.Current())
			ok = moveFSM(userFsm, m.Postback.Payload)
			if ok {
				glog.Infof("successful transition from '%s' with msg: '%s' to '%s'. Available transitions are: %+v", oldState.Name, m.Postback.Payload, userFsm.Current(), userFsm.AvailableTransitions())
			} else {
				glog.Warningf("failed to make transition from '%s' with msg '%s'. Available transitions are: %+v", oldState.Name, m.Postback.Payload, userFsm.AvailableTransitions())
			}

			respondToUser(*userFsm, m.Sender.ID)
		}
	}
	return nil
}

func resetFSM(userFsm map[string]*fsm.FSM, userid string) {
	userFsm[userid] = statemachine.ChatBotFSM()
}

func isVerifyRequest(w http.ResponseWriter, r *http.Request) bool {
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

func respondToUser(userFsm fsm.FSM, recipient string) error {
	// Send raw message with long explanation
	currentState := statemachine.StateFromString(userFsm.Current())
	msg := currentState.Message
	sendRawMessage(recipient, msg)

	// Create postback with options to choose from next
	events := transitionsToEvents(userFsm.AvailableTransitions())
	events = statemachine.Sorted(events)
	buttons := eventsToPostbackButtons(events)
	elements := getPostbackElements("What's next?", "Tap to answer", buttons)
	// Get consistent button order
	payload := getPostbackPaylod(recipient, elements)
	sendPostBackMessage(recipient, payload)
	return nil
}

// Given a user state machine and a message, try to make a transition and create a response
func moveFSM(userFsm *fsm.FSM, event string) bool {
	// If transition is allowed
	if userFsm.Can(event) {
		userFsm.Event(event)
		return true
	} else {
		return false
	}
}

func transitionsToEvents(transitions []string) []statemachine.Event {
	res := []statemachine.Event{}
	for _, t := range transitions {
		res = append(res, statemachine.EventFromString(t))
	}
	return res
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

func eventsToPostbackButtons(events []statemachine.Event) []models.MessageWithPostbackButton {
	transitions := []string{}
	for _, e := range events {
		transitions = append(transitions, e.Name)
	}
	return transitionsToPostbackButtons(transitions)
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
