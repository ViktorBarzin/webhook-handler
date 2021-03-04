package statemachine

import "sort"

type Event struct {
	// Internal event name
	Name string
	// Message shown to user
	Message string
	// order of events shown to user
	orderID int
}

const (
	InvalidEventName         = "Invalid"
	BackEventName            = "Back"
	GetStartedEventName      = "GetStarted"
	HelpEventName            = "Help"
	ShowBlogIntoEventName    = "ShowBlogInfo"
	ShowF1InfoEventName      = "ShowF1Info"
	ShowGrafanaInfoEventName = "ShowGrafanaInfo"
	ShowHackmdInfoEventName  = "ShowHackmdInfo"
)

var (
	// Map of all available transitions (events)
	// Higher orderID, lower priority i.e 0 will be show first, 100 last
	Events = map[string]Event{
		InvalidEventName: {
			Name:    InvalidEventName,
			Message: "Invalid Event",
			orderID: 100, // lowest priority
		},
		BackEventName: {
			Name:    BackEventName,
			Message: "Back",
			orderID: 95,
		},
		GetStartedEventName: {
			Name:    GetStartedEventName,
			Message: "Get Started!",
			orderID: 10,
		},
		HelpEventName: {
			Name:    HelpEventName,
			Message: "Help",
			orderID: 94,
		},
		ShowBlogIntoEventName: {
			Name:    ShowBlogIntoEventName,
			Message: "Blog info",
			orderID: 11,
		},
		ShowF1InfoEventName: {
			Name:    ShowF1InfoEventName,
			Message: "F1 info",
			orderID: 20,
		},
		ShowGrafanaInfoEventName: {
			Name:    ShowGrafanaInfoEventName,
			Message: "Dashboards",
			orderID: 12,
		},
		ShowHackmdInfoEventName: {
			Name:    ShowHackmdInfoEventName,
			Message: "Document collab tool",
			orderID: 13,
		},
	}
)

func EventFromString(s string) Event {
	if _, ok := Events[s]; !ok {
		return Events[InvalidEventName]
	}
	return Events[s]
}

// Return Events sorted by their message order
func Sorted(events []Event) []Event {
	values := []Event{}
	for _, v := range events {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].orderID < values[j].orderID
	})
	return values
}
