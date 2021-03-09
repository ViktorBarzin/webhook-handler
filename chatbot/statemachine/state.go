package statemachine

type State struct {
	// State name for FSM
	Name string `yaml:"id"`
	// Message send to user at this state
	Message string `yaml:"message"`
}

func (s State) String() string {
	return s.Name
}

func NewState(name, message string) State {
	return State{Name: name, Message: message}
}
