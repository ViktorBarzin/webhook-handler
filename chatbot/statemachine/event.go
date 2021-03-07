package statemachine

import "sort"

type Event struct {
	// Internal event name
	Name string `yaml:"id"`
	// Message shown to user
	Message string `yaml:"message"`
	// order of events shown to user
	OrderID int `yaml:"orderID"`
}

// const (
// 	InvalidEventName         = "Invalid"
// 	BackEventName            = "Back"
// GetStartedEventName = "GetStarted"

// 	HelpEventName            = "Help"
// 	ShowBlogIntoEventName    = "ShowBlogInfo"
// 	ShowF1InfoEventName      = "ShowF1Info"
// 	ShowGrafanaInfoEventName = "ShowGrafanaInfo"
// 	ShowHackmdInfoEventName  = "ShowHackmdInfo"
// 	ShowPrivatebinEventName  = "ShowPrivatebinInfo"
// 	ResetEventName           = "Reset"
// )

// var (
// 	// Map of all available transitions (events)
// 	// Higher orderID, lower priority i.e 0 will be show first, 100 last
// 	Events = map[string]Event{
// 		InvalidEventName: {
// 			Name:    InvalidEventName,
// 			Message: "Invalid Event",
// 			orderID: 100, // lowest priority
// 		},
// 		BackEventName: {
// 			Name:    BackEventName,
// 			Message: "Back",
// 			orderID: 95,
// 		},
// 		GetStartedEventName: {
// 			Name:    GetStartedEventName,
// 			Message: "Get Started!",
// 			orderID: 10,
// 		},
// 		HelpEventName: {
// 			Name:    HelpEventName,
// 			Message: "Help",
// 			orderID: 94,
// 		},
// 		ShowBlogIntoEventName: {
// 			Name:    ShowBlogIntoEventName,
// 			Message: "Blog info",
// 			orderID: 11,
// 		},
// 		ShowF1InfoEventName: {
// 			Name:    ShowF1InfoEventName,
// 			Message: "F1 info",
// 			orderID: 20,
// 		},
// 		ShowGrafanaInfoEventName: {
// 			Name:    ShowGrafanaInfoEventName,
// 			Message: "Dashboards",
// 			orderID: 12,
// 		},
// 		ShowPrivatebinEventName: {
// 			Name:    ShowPrivatebinEventName,
// 			Message: "Create paste",
// 			orderID: 13,
// 		},
// 		ShowHackmdInfoEventName: {
// 			Name:    ShowHackmdInfoEventName,
// 			Message: "Document collab tool",
// 			orderID: 14,
// 		},
// 		ResetEventName: {
// 			Name:    ResetEventName,
// 			Message: "Reset conversation",
// 			orderID: 99,
// 		},
// 	}
// )

// func EventFromString(s string) Event {
// 	if _, ok := Events[s]; !ok {
// 		return Events[InvalidEventName]
// 	}
// 	return Events[s]
// }

// Sorted returns Events sorted by their message order
func Sorted(events []Event) []Event {
	values := []Event{}
	for _, v := range events {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].OrderID < values[j].OrderID
	})
	return values
}

func NewEvent(name, message string, orderID int) Event {
	return Event{Name: name, Message: message, OrderID: orderID}
}
