package statemachine

import "github.com/looplab/fsm"

func ChatBotFSM() *fsm.FSM {
	return fsm.NewFSM(InitialStateName,
		[]fsm.EventDesc{
			// Get Started
			{
				Name: GetStartedEventName,
				Src:  []string{InitialStateName},
				Dst:  HelloStateName,
			},
			// Help Event
			{
				Name: HelpEventName,
				Src:  []string{InitialStateName},
				Dst:  InitialStateName,
			},
			{
				Name: HelpEventName,
				Src:  []string{HelloStateName},
				Dst:  HelloStateName,
			},
			{
				Name: HelpEventName,
				Src:  []string{F1StateName},
				Dst:  F1StateName,
			},
			// Reset
			{
				Name: ResetEventName,
				Src:  []string{HelloStateName},
				Dst:  HelloStateName,
			},
			{
				Name: ResetEventName,
				Src:  []string{F1StateName},
				Dst:  HelloStateName,
			},
			// Show F1 info
			{
				Name: ShowF1InfoEventName,
				Src:  []string{HelloStateName},
				Dst:  F1StateName,
			},
		},
		map[string]fsm.Callback{})
}

func Main() {
	kek := ChatBotFSM()

	println(fsm.Visualize(kek))
	// err := kek.Event("bur")
	// println(kek.Current())
	// println(err.Error())
}
