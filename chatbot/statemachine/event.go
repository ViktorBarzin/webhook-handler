package statemachine

type Event struct {
	Name    string
	Message string
}

const (
	InvalidEventName    = "Invalid"
	ResetEventName      = "Reset"
	GetStartedEventName = "GetStarted"
	HelpEventName       = "Help"
	ShowF1InfoEventName = "ShowF1Info"
)

var (
	InvalidEvent Event = Event{
		Name:    InvalidEventName,
		Message: "Invalid Event",
	}
	Reset Event = Event{
		Name:    ResetEventName,
		Message: "Reset",
	}
	GetStarted Event = Event{
		Name:    GetStartedEventName,
		Message: "Get Started!",
	}
	Help Event = Event{
		Name:    HelpEventName,
		Message: "Help",
	}
	ShowF1Info Event = Event{
		Name:    ShowF1InfoEventName,
		Message: "Show F1 services information",
	}
)

func EventFromString(s string) Event {
	switch s {
	case InvalidEventName:
		return InvalidEvent
	case ResetEventName:
		return Reset
	case GetStartedEventName:
		return GetStarted
	case HelpEventName:
		return Help
	case ShowF1InfoEventName:
		return ShowF1Info
	default:
		return InvalidEvent
	}
}
