package chatbot

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/viktorbarzin/webhook-handler/chatbot/auth"
	"github.com/viktorbarzin/webhook-handler/chatbot/fbapi"
	"github.com/viktorbarzin/webhook-handler/chatbot/statemachine"
)

type ApprovalRequestPayload struct {
	ID    string    `json:"id"`
	From  auth.User `json:"from"`
	CmdID string    `json:"cmdID"`
	// What    auth.Command  `json:"what"`
	Payload string        `json:"payload"`
	State   ApprovalState `json:"state"`
}

type ApprovalState int

const (
	ApprovalStatePending = ApprovalState(iota)
	ApprovalStateAccepted
	ApprovalStateRejected
)

func (a ApprovalState) String() string {
	switch a {
	case ApprovalStateAccepted:
		return "Accepted"
	case ApprovalStateRejected:
		return "Rejected"
	case ApprovalStatePending:
		return "Pending"
	default:
		return "Unknown"
	}
}

func isApprovalRequest(payload []byte) bool {
	var a ApprovalRequestPayload
	return json.Unmarshal(payload, &a) == nil
}

func (c *ChatbotHandler) cmdFromId(id string) (auth.Command, error) {
	var res auth.Command
	for _, cmd := range c.RBACConfig.Commands {
		if cmd.ID == id {
			res = cmd
		}
	}
	if reflect.DeepEqual(res, auth.Command{}) {
		return res, fmt.Errorf("failed to find command with id %s", id)
	}
	return res, nil
}

// send request to all users in the `approvedBy` role
func (c *ChatbotHandler) sendRequestApprovalRequest(from auth.User, what auth.Command, payload string) error {
	if len(c.RBACConfig.UsersInRole(what.ApprovedBy)) == 0 {
		return fmt.Errorf("no users can approve command '%s': '%s'", what.PrettyName, what.CMD)
	}
	requestMsg := fmt.Sprintf("User '%s'(ID: %s) wants to execute '%s' with input: '%s'", from.Name, from.ID, what.PrettyName, payload)

	acceptPayload, err := serializeApprovalRequest(acceptApprovalRequestPayload(from, what, payload))
	if err != nil {
		return errors.Wrapf(err, "failed to get serialized accept payload")
	}

	rejectPayload, err := serializeApprovalRequest(rejectApprovalRequestPayload(from, what, payload))
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
		err := fbapi.SendRawMessage(u.ID, requestMsg)
		if err != nil {
			glog.Warningf("failed to send auth request for '%+v' to user %+v", what, u)
			continue
		}
		payload := getPostbackPayload(u.ID, elements)
		if err := fbapi.SendPostBackMessage(u.ID, payload); err != nil {
			glog.Warningf("failed to send postback message '%+v' to user '%+v'; Error: %s", payload, u, err.Error())
		}
	}
	return nil
}

// SendApprovalRequestUpdateNotification sends notification to the creator of the request for its status
func (c *ChatbotHandler) SendApprovalRequestUpdateNotification(r ApprovalRequestPayload, moderator auth.User) error {
	cmd, err := c.cmdFromId(r.CmdID)
	if err != nil {
		return errors.Wrapf(err, "failed to get cmd from id")
	}
	requestMsg := fmt.Sprintf("Your request to execute '%s' with input '%s' has been %s by %s", cmd.PrettyName, r.Payload, r.State.String(), moderator.Name)

	if err := fbapi.SendRawMessage(r.From.ID, requestMsg); err != nil {
		return errors.Wrapf(err, "failed to notify request sender about the status of their request")
	}

	moderatorMsg := fmt.Sprintf("Successfully notified %s (ID: %s) about your decision on '%s'. Decision outcome: %s", r.From.Name, r.From.ID, cmd.PrettyName, r.State.String())
	err = fbapi.SendRawMessage(moderator.ID, moderatorMsg)
	if err != nil {
		return errors.Wrapf(err, "failed to notify moderator about the success of their request approval/rejection")
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

func NewApprovalRequest(from auth.User, what auth.Command, state ApprovalState, userInput string) ApprovalRequestPayload {
	return ApprovalRequestPayload{ID: uuid.New().URN(), From: from, CmdID: what.ID, State: state, Payload: userInput}
}

func DeserializeApprovalRequest(p []byte) (ApprovalRequestPayload, error) {
	var a ApprovalRequestPayload
	err := json.Unmarshal(p, &a)
	if err != nil {
		return ApprovalRequestPayload{}, errors.Wrapf(err, "failed to deserialize approval request from payload: %s", string(p))
	}
	return a, nil
}

func acceptApprovalRequestPayload(from auth.User, what auth.Command, userInput string) ApprovalRequestPayload {
	return NewApprovalRequest(from, what, ApprovalStateAccepted, userInput)
}

func rejectApprovalRequestPayload(from auth.User, what auth.Command, userInput string) ApprovalRequestPayload {
	return NewApprovalRequest(from, what, ApprovalStateRejected, userInput)
}
