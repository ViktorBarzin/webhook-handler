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
	GrafanaStateName = "Grafana"
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
			Message: "To visit my website go to https://viktorbarzin.me",
		},
		F1StateName: {
			Name:    F1StateName,
			Message: "To watch F1 streams go to http://f1.viktorbarzin.me",
		},
		GrafanaStateName: {
			Name:    GrafanaStateName,
			Message: "To see my infrastructure dashboards go to https://grafana.viktorbarzin.me/dashboards",
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
