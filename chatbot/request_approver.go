package chatbot

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/viktorbarzin/webhook-handler/chatbot/auth"
	"github.com/viktorbarzin/webhook-handler/chatbot/fbapi"
	"github.com/viktorbarzin/webhook-handler/chatbot/statemachine"
)

type ApprovalRequestPayload struct {
	From    auth.User     `json:"from"`
	What    auth.Command  `json:"what"`
	Payload string        `json:"payload"`
	State   ApprovalState `json:"state"`
}

type ApprovalState int

const (
	ApprovalStatePending = ApprovalState(iota)
	ApprovalStateAccepted
	ApprovalStateRejected
)

func isApprovalRequest(payload []byte) bool {
	var a ApprovalRequestPayload
	return json.Unmarshal(payload, &a) == nil
}

// send request to all users in the `approvedBy` role
func (c *ChatbotHandler) sendRequestApprovalRequest(from auth.User, what auth.Command, payload string) error {
	if len(c.RBACConfig.UsersInRole(what.ApprovedBy)) == 0 {
		return fmt.Errorf("no users can approve command '%s': '%s'", what.PrettyName, what.CMD)
	}
	requestMsg := fmt.Sprintf("User '%s'(ID: %s) wants to execute '%s' with input: '%s'", from.Name, from.ID, what.PrettyName, payload)

	acceptPayload, err := serializeApprovalRequest(acceptApprovalRequestPayload(from, what))
	if err != nil {
		return errors.Wrapf(err, "failed to get serialized accept payload")
	}

	rejectPayload, err := serializeApprovalRequest(rejectApprovalRequestPayload(from, what))
	if err != nil {
		return errors.Wrapf(err, "failed to get serialized reject payload")
	}

	// Get accept/reject buttons
	events := []statemachine.Event{
		{
			Name:    acceptPayload,
			Message: "Approve",
		},
		{
			Name:    rejectPayload,
			Message: "Deny",
		},
	}
	buttons := eventsToPostbackButtons(events)
	elements := getPostbackElements("Select action for this request", "Tap to answer", buttons)
	// send request to all users with this role
	for _, u := range c.RBACConfig.UsersInRole(what.ApprovedBy) {
		_, err := fbapi.SendRawMessage(u.ID, requestMsg)
		if err != nil {
			glog.Warningf("failed to send auth request for '%+v' to user %+v", what, u)
			continue
		}
		payload := getPostbackPayload(u.ID, elements)
		if _, err := fbapi.SendPostBackMessage(u.ID, payload); err != nil {
			glog.Warningf("failed to send postback message '%+v' to user '%+v'; Error: %s", payload, u, err.Error())
		}
	}
	return nil
}

func serializeApprovalRequest(a ApprovalRequestPayload) (string, error) {
	serialized, err := json.Marshal(a)
	if err != nil {
		return "", errors.Wrapf(err, "failed to serialize approval request: %+v", a)
	}
	return string(serialized), nil

}

func NewApprovalRequest(from auth.User, what auth.Command, state ApprovalState) ApprovalRequestPayload {
	return ApprovalRequestPayload{From: from, What: what, State: state}
}

func acceptApprovalRequestPayload(from auth.User, what auth.Command) ApprovalRequestPayload {
	return NewApprovalRequest(from, what, ApprovalStateAccepted)
}

func rejectApprovalRequestPayload(from auth.User, what auth.Command) ApprovalRequestPayload {
	return NewApprovalRequest(from, what, ApprovalStateRejected)
}
