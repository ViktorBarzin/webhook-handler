package statemachine

type Event struct {
	Name    string
	Message string
}

const (
	InvalidEventName      = "Invalid"
	BackEventName         = "Back"
	GetStartedEventName   = "GetStarted"
	HelpEventName         = "Help"
	ShowBlogIntoEventName = "ShowBlogInfo"
	ShowF1InfoEventName   = "ShowF1Info"
)

var (
	InvalidEvent Event = Event{
		Name:    InvalidEventName,
		Message: "Invalid Event",
	}
	Back Event = Event{
		Name:    BackEventName,
		Message: "Back",
	}
	GetStarted Event = Event{
		Name:    GetStartedEventName,
		Message: "Get Started!",
	}
	Help Event = Event{
		Name:    HelpEventName,
		Message: "Help",
	}
	ShowBlogInfo Event = Event{
		Name:    ShowBlogIntoEventName,
		Message: "Blog info",
	}
	ShowF1Info Event = Event{
		Name:    ShowF1InfoEventName,
		Message: "F1 info",
	}
)

func EventFromString(s string) Event {
	switch s {
	case InvalidEventName:
		return InvalidEvent
	case BackEventName:
		return Back
	case GetStartedEventName:
		return GetStarted
	case HelpEventName:
		return Help
	case ShowBlogIntoEventName:
		return ShowBlogInfo
	case ShowF1InfoEventName:
		return ShowF1Info
	default:
		return InvalidEvent
	}
}
