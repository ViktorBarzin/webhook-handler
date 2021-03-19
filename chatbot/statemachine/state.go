package statemachine

import (
	"viktorbarzin/webhook-handler/chatbot/auth"

	"github.com/viktorbarzin/gorbac"
)

type State struct {
	// State name for FSM
	Name string `yaml:"id"`
	// Message send to user at this state
	Message        string                 `yaml:"message"`
	Permissions    []gorbac.StdPermission `yaml:"permissions"`
	Commands       []auth.Command         `yaml:"commands"`
	DefaultHandler auth.Command           `yaml:"defaultHandler"`
}

func NewState(name, message string) State {
	return State{Name: name, Message: message}
}
