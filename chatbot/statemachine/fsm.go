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
	FSM       *fsm.FSM
	EventDesc []fsm.EventDesc `yaml:"statemachine"`
	States    []State         `yaml:"states"`
	Events    []Event         `yaml:"events"`
}

func ChatBotFSM(configFile string) (*FSMWithStatesAndEvents, error) {

	fileBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file %s", configFile)
	}
	dec := yaml.NewDecoder(bytes.NewReader(fileBytes))

	// Decode FSM
	var f *FSMWithStatesAndEvents = nil

	foundRBACSpec := false
	// try to find rbac config in the config file
	for {
		err = dec.Decode(&f)
		if err != nil {
			break
		}
		if f != nil {
			foundRBACSpec = true
			break
		}
	}
	if !foundRBACSpec || err != nil {
		return nil, errors.Errorf("did not find valid FSM config in file %s. Err: %s", configFile, err.Error())
	}
	f.FSM = fsm.NewFSM("Initial", f.EventDesc, map[string]fsm.Callback{})
	glog.Infof("successfully parsed config file into fsm %+v", f)
	return f, nil
}

func (f FSMWithStatesAndEvents) Current() State {
	res := State{}
	for _, s := range f.States {
		if s.Name == f.FSM.Current() {
			res = s
		}
	}
	return res
}

func (f FSMWithStatesAndEvents) AvailableTransitions() []Event {
	transitions := f.FSM.AvailableTransitions()

	// eventName -> Event
	allEvents := map[string]Event{}
	for _, e := range f.Events {
		allEvents[e.Name] = e
	}

	res := []Event{}
	// pick transitions
	for _, t := range transitions {
		if e, ok := allEvents[t]; ok {
			res = append(res, e)
		}
	}
	return res
}
