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
