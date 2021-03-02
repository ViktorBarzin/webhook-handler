package statemachine

// type State int

type State struct {
	// State name for FSM
	Name string
	// Message send to user at this state
	Message string
}

// State name literals
const (
	InvalidStateName = "Invalid"
	InitialStateName = "Initial"
	HelloStateName   = "Hello"
	BlogStateName    = "Blog"
	F1StateName      = "F1"
)

var (
	InvalidState State = State{
		Name:    InvalidStateName,
		Message: "Oops, I didn't quite get that. Please use the buttons only.",
	}
	Initial State = State{
		Name:    InitialStateName,
		Message: "Let's get started",
	}
	Hello State = State{
		Name:    HelloStateName,
		Message: `How can I help?`,
	}
	Blog State = State{
		Name:    BlogStateName,
		Message: "To visit my website go to \"https://viktorbarzin.me\"",
	}
	F1 State = State{
		Name:    F1StateName,
		Message: `To watch F1 streams go to "http://f1.viktorbarzin.me"`,
	}
)

// StateFromString creates state from a string
func StateFromString(s string) State {
	switch s {
	case InvalidStateName:
		return InvalidState
	case InitialStateName:
		return Initial
	case HelloStateName:
		return Hello
	case BlogStateName:
		return Blog
	case F1StateName:
		return F1
	default:
		return InvalidState
	}
}

func (s State) String() string {
	return s.Name
}
