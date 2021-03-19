package statemachine

import (
	"fmt"
	"regexp"
	"strings"
	"viktorbarzin/webhook-handler/chatbot/auth"
	"viktorbarzin/webhook-handler/chatbot/executor"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

/* Special states are states which are backed by code implementation.
They are not fully defined by the config file.

Special states are like normal states, however they accept arbitrary input which is passed to a callback function.
This input is passed to the callback function iff no transition can be made.
This function can run some side effect but cannot change the state whatsoever.

An example for such state is one that needs to execute some code or accept arbitrary input.
E.g: VPN state would like to accept any input for the public key and run some os commands.*/

// SpecialStateType should be defined in the state itself
type SpecialStateType string

// Defined special state types. Make sure you define new ones in the map below
const (
	VPNStateType = "vpn"
)

var (
	SpecialStateTypeCallback map[SpecialStateType]func(string) (string, error) = map[SpecialStateType]func(string) (string, error){
		VPNStateType: VPNStateTypeHandler,
	}

	vpnFriendlyNameRegex = regexp.MustCompile(`(\w| ){1,40}`)
	vpnPubKeyRegex       = regexp.MustCompile(`[-A-Za-z0-9+=]{1,50}|=[^=]|={3,}`)
)

// VPNStateTypeHandler is called with any user input
func VPNStateTypeHandler(event string) (string, error) {
	glog.Infof("VPN HANDLERRR")
	split := strings.Split(event, " ")
	if len(split) != 2 {
		return "", fmt.Errorf("invalid format, please provide message in the format: 'friendly_name your_pubic_key' (cert friendly name, <SPACE> your public key)")
	}
	friendlyName := split[0]
	pubKey := split[1]
	if found := vpnFriendlyNameRegex.FindString(friendlyName); found == "" {
		return "", fmt.Errorf("VPN friendly name must be '%s'. Got %s", vpnFriendlyNameRegex.String(), friendlyName)
	}
	if found := vpnPubKeyRegex.FindString(pubKey); found == "" {
		return "", fmt.Errorf("Invalid public key found. Expected key to match '%s', got key: '%s'", vpnPubKeyRegex.String(), pubKey)
	}

	// Command args are escaped upon execution
	cmd := fmt.Sprintf("%s -use-case vpn -vpn-client-name %s -vpn-pub-key %s", executor.InfraCli, friendlyName, pubKey)
	glog.Infof("running command to generate client config: '%s'", cmd)

	output, err := executor.Execute(auth.Command{PrettyName: "Add Wireguard client config", CMD: cmd}, "kek")
	if err != nil {
		return "", errors.Wrapf(err, "creating client config failed")
	}
	return output, nil
}
