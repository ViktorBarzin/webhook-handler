package statemachine

import (
	"github.com/viktorbarzin/gorbac"
)

type State struct {
	// State name for FSM
	Name string `yaml:"id"`
	// Message send to user at this state
	Message     string                 `yaml:"message"`
	Permissions []gorbac.StdPermission `yaml:"permissions"`
}

func NewState(name, message string) State {
	return State{Name: name, Message: message}
}
