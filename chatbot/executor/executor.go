package executor

import (
	"os/exec"
	"strings"
	"viktorbarzin/webhook-handler/chatbot/auth"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	InfraCli = "infra_cli"
)

// Execute runs the given command blocking
func Execute(cmd auth.Command) (string, error) {
	glog.Infof("executing '%s': '%s'", cmd.PrettyName, cmd.CMD)
	cmdArgs := strings.Split(cmd.CMD, " ")
	if len(cmdArgs) == 0 {
		// nothing to execute
		glog.Infof("skipping executing empty command: %+v", cmd)
		return "", nil
	}
	binary, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		return "", errors.Wrapf(err, "failed to find binary")
	}

	c := exec.Command(binary, cmdArgs[1:]...)
	glog.Infof(strings.Repeat("-", 40))
	glog.Infof("executing: '%s'", c.String())
	output, err := c.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to execute cmd: %s", c.String())
	}
	glog.Infof("cmd combined output: %s", string(output))
	glog.Infof(strings.Repeat("-", 40))
	return string(output), nil
}
