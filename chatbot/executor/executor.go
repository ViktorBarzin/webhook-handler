package executor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/viktorbarzin/webhook-handler/chatbot/auth"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	InfraCli = "infra_cli"
	bashCli  = "/bin/sh"
)

// Execute runs the given command blocking
func Execute(cmd auth.Command, input string) (string, error) {
	bashCmd := fmt.Sprintf("echo '%s' | while read -r line; do\n %s\n done", input, cmd.CMD)
	c := exec.Command("/bin/sh", "-c", bashCmd)
	glog.Infof(strings.Repeat("-", 40))
	glog.Infof("Command: %+v, input: %s", cmd, input)
	// glog.Infof("executing: '%s'", c.String())
	output, err := c.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to execute cmd: '%s' with error '%s'.\n Command output: %s", cmd.PrettyName, err.Error(), string(output))
	}
	glog.Infof("cmd combined output: %s", string(output))
	glog.Infof(strings.Repeat("-", 40))
	return string(output), nil
}
