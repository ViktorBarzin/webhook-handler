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
				Src:  []string{BlogStateName},
				Dst:  BlogStateName,
			},
			{
				Name: HelpEventName,
				Src:  []string{F1StateName},
				Dst:  F1StateName,
			},
			{
				Name: HelpEventName,
				Src:  []string{GrafanaStateName},
				Dst:  GrafanaStateName,
			},
			// Back
			{
				Name: BackEventName,
				Src:  []string{BlogStateName, F1StateName, GrafanaStateName, HackmdStateName},
				Dst:  HelloStateName,
			},
			// Show blog info
			{
				Name: ShowBlogIntoEventName,
				Src:  []string{HelloStateName},
				Dst:  BlogStateName,
			},

			// Show F1 info
			{
				Name: ShowF1InfoEventName,
				Src:  []string{HelloStateName},
				Dst:  F1StateName,
			},
			// Show Grafana info
			{
				Name: ShowGrafanaInfoEventName,
				Src:  []string{HelloStateName},
				Dst:  GrafanaStateName,
			},
			// Show hackmd info
			{
				Name: ShowHackmdInfoEventName,
				Src:  []string{HelloStateName},
				Dst:  HackmdStateName,
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
