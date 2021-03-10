package auth

import (
	"bytes"
	"io/ioutil"
	"reflect"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func NewRBACConfig(configFile string) (RBACConfig, error) {
	fileBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return RBACConfig{}, errors.Wrapf(err, "failed to read config file %s", configFile)
	}
	dec := yaml.NewDecoder(bytes.NewReader(fileBytes))

	// Decode FSM
	var rbacYaml RBACConfig

	foundRBACSpec := false
	// try to find rbac config in the config file
	for {
		err = dec.Decode(&rbacYaml)
		if err != nil {
			break
		}
		if !reflect.DeepEqual(rbacYaml, RBACConfig{}) {
			foundRBACSpec = true
			break
		}
	}
	if !foundRBACSpec || err != nil {
		return RBACConfig{}, errors.Errorf("did not find valid RBAC config in file %s. Err: %s", configFile, err.Error())
	}
	glog.Infof("RBAC config: %+v", rbacYaml)
	return rbacYaml, nil
}
