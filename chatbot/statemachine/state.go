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
	InvalidStateName    = "Invalid"
	InitialStateName    = "Initial"
	HelloStateName      = "Hello"
	BlogStateName       = "Blog"
	F1StateName         = "F1"
	GrafanaStateName    = "Grafana"
	HackmdStateName     = "Hackmd"
	PrivatebinStateName = "Privatebin"
)

var (
	States = map[string]State{
		InvalidEventName: {
			Name:    InvalidStateName,
			Message: "Oops, I didn't quite get that. Please use the buttons only.",
		},
		InitialStateName: {
			Name:    InitialStateName,
			Message: "Let's get started",
		},
		HelloStateName: {
			Name:    HelloStateName,
			Message: `How can I help?`,
		},
		BlogStateName: {
			Name:    BlogStateName,
			Message: "I have a website where I casually blog on various tech topics. To visit my website go to https://viktorbarzin.me",
		},
		F1StateName: {
			Name:    F1StateName,
			Message: "I have an F1 streaming site, where you can watch F1 streams without annoying pop-ups and ads. \n To watch F1 streams go to http://f1.viktorbarzin.me",
		},
		GrafanaStateName: {
			Name:    GrafanaStateName,
			Message: "I have some pretty dashboards about my infrastructure. Available at https://grafana.viktorbarzin.me/dashboards",
		},
		HackmdStateName: {
			Name:    HackmdStateName,
			Message: "Document collaboration tool. Similar to Google Docs. Available at https://hackmd.viktorbarzin.me",
		},
		PrivatebinStateName: {
			Name:    PrivatebinStateName,
			Message: "Share pastes securely. Available at https://pb.viktorbarzin.me",
		},
	}
)

// StateFromString creates state from a string
func StateFromString(s string) State {
	if _, ok := States[s]; !ok {
		return States[InvalidStateName]
	}
	return States[s]
}

func (s State) String() string {
	return s.Name
}
