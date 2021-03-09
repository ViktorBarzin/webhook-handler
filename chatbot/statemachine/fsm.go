package statemachine

import (
	"bytes"
	"io/ioutil"

	"github.com/golang/glog"
	"github.com/looplab/fsm"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type FSMWithStatesAndEvents struct {
	FSM    *fsm.FSM
	States map[string]State
	Events map[string]Event
}

type EventDescYaml struct {
	Fsm []struct {
		Name      string   `yaml:"name"`
		SrcState  []string `yaml:"srcState"`
		DestState string   `yaml:"destState"`
	}
}

type StateYaml struct {
	States []State
}

type EventYaml struct {
	Events []Event
}

func ChatBotFSM(configFile string) (*FSMWithStatesAndEvents, error) {
	fileBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file %s", configFile)
	}
	fsm, err := yamlToFSM(fileBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create FSM from contents of file %s", configFile)
	}
	return fsm, nil
}

func yamlToFSM(yamlSpec []byte) (*FSMWithStatesAndEvents, error) {
	dec := yaml.NewDecoder(bytes.NewReader(yamlSpec))

	// Decode FSM
	var eventDescYaml EventDescYaml
	if err := dec.Decode(&eventDescYaml); err != nil {
		return nil, errors.Wrapf(err, "failed to decode FSM")
	}
	glog.Infof("Successfully decoded FSM from config file: %+v", eventDescYaml)

	// Decode States
	var states StateYaml
	if err := dec.Decode(&states); err != nil {
		return nil, errors.Wrapf(err, "failed to decode states list")
	}
	glog.Infof("Successfully decoded states from config file: %+v", states)

	var events EventYaml
	if err := dec.Decode(&events); err != nil {
		return nil, errors.Wrapf(err, "failed to decode events list")
	}
	glog.Infof("Successfully decoded events from config file: %+v", events)

	eventDes := eventDescYamlToEventDesc(eventDescYaml)
	glog.Infof("AAAAA\n\n%+v\n\n", eventDes)

	statesMap := map[string]State{}
	for _, s := range states.States {
		statesMap[s.Name] = NewState(s.Name, s.Message)
	}
	eventsMap := map[string]Event{}
	for _, e := range events.Events {
		eventsMap[e.Name] = NewEvent(e.Name, e.Message, e.OrderID)
	}
	return &FSMWithStatesAndEvents{
		FSM:    fsm.NewFSM("Initial", eventDes, map[string]fsm.Callback{}),
		States: statesMap,
		Events: eventsMap,
	}, nil
}

func eventDescYamlToEventDesc(m EventDescYaml) []fsm.EventDesc {
	res := []fsm.EventDesc{}
	for _, e := range m.Fsm {
		res = append(res, fsm.EventDesc{Name: e.Name, Src: e.SrcState, Dst: e.DestState})
	}
	return res
}

func (f FSMWithStatesAndEvents) Current() State {
	return f.States[f.FSM.Current()]
}

func (f FSMWithStatesAndEvents) AvailableTransitions() []Event {
	transitions := f.FSM.AvailableTransitions()
	res := []Event{}
	for _, t := range transitions {
		res = append(res, f.Events[t])
	}
	return res
}
