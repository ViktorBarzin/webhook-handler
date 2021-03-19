package auth

import (
	"bytes"
	"io/ioutil"
	"reflect"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/viktorbarzin/gorbac"
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

	// Add guest user
	rbacYaml.Users = append(rbacYaml.Users, GuestUser())

	// Remove duplicates
	rbacYaml.Users = uniqueUsers(rbacYaml.Users)
	rbacYaml.Commands = uniqueCommands(rbacYaml.Commands)
	rbacYaml.Permissions = uniquePermissions(rbacYaml.Permissions)
	rbacYaml.Roles = uniqueRoles(rbacYaml.Roles)
	rbacYaml.Groups = uniqueGroups(rbacYaml.Groups)

	rbac := gorbac.New()
	initRBAC(rbac, rbacYaml.Users)
	rbacYaml.RBAC = rbac

	// p := rbacYaml.Permissions[0]
	// ok := rbac.IsGranted("viktor-fbid", p, nil)
	// glog.Infof("Is granted: %t", ok)

	glog.Infof("RBAC config: %+v", rbacYaml)
	return rbacYaml, nil
}

func uniqueUsers(users []User) []User {
	check := map[string]bool{}
	for _, u := range users {
		check[u.ID] = true
	}
	res := []User{}
	for _, v := range users {
		if _, ok := check[v.ID]; ok {
			res = append(res, v)
		}
	}
	return res
}

func uniqueCommands(commands []Command) []Command {
	check := map[string]bool{}
	for _, u := range commands {
		check[u.ID] = true
	}
	res := []Command{}
	for _, v := range commands {
		if _, ok := check[v.ID]; ok {
			res = append(res, v)
		}
	}
	return res
}

func uniquePermissions(permissions []gorbac.StdPermission) []gorbac.StdPermission {
	check := map[string]bool{}
	for _, u := range permissions {
		check[u.ID()] = true
	}
	res := []gorbac.StdPermission{}
	for _, v := range permissions {
		if _, ok := check[v.ID()]; ok {
			res = append(res, v)
		}
	}
	return res
}

func uniqueRoles(roles []Role) []Role {
	check := map[string]bool{}
	for _, u := range roles {
		check[u.Name] = true
	}
	res := []Role{}
	for _, v := range roles {
		if _, ok := check[v.Name]; ok {
			res = append(res, v)
		}
	}
	return res
}

func uniqueGroups(groups []Group) []Group {
	check := map[string]bool{}
	for _, u := range groups {
		check[u.Name] = true
	}
	res := []Group{}
	for _, v := range groups {
		if _, ok := check[v.Name]; ok {
			res = append(res, v)
		}
	}
	return res
}

func ToPermissions(ps []gorbac.StdPermission) []gorbac.Permission {
	res := make([]gorbac.Permission, len(ps))
	for i, v := range ps {
		res[i] = v
	}
	return res
}

func initRBAC(rbac *gorbac.RBAC, users []User) {
	// Init roles and permissions
	for _, u := range users {
		userRole := gorbac.NewStdRole(u.ID)
		rbac.Add(userRole)
		for _, r := range u.Roles {
			role := gorbac.NewStdRole(r.Name)
			for _, p := range r.Permissions {
				perm := gorbac.NewStdPermission(p.ID())
				role.Assign(perm)
			}
			rbac.Add(role)
			rbac.SetParent(userRole.ID(), role.ID())
		}
		for _, g := range u.Groups {
			groupRole := gorbac.NewStdRole(g.Name)
			rbac.Add(groupRole)
			for _, r := range g.Roles {
				gRole := gorbac.NewStdRole(r.Name)
				for _, p := range r.Permissions {
					perm := gorbac.NewStdPermission(p.ID())
					gRole.Assign(perm)
				}
				rbac.Add(gRole)
				rbac.SetParent(groupRole.ID(), gRole.ID())
			}
			rbac.SetParent(userRole.ID(), groupRole.ID())
		}
	}
}

func (c RBACConfig) WhoAmI(userId string) User {
	var res User
	for _, u := range c.Users {
		// store guest user
		if u.ID == GuestUserID {
			res = u
		}
		if u.ID == userId {
			res = u
			break
		}
	}
	return res
}

func (c RBACConfig) IsAllowed(user User, p gorbac.Permission) bool {
	// glog.Infof("Checking: Have: %+v, Perm: %+v ", userID, p)
	if c.RBAC.IsGranted(user.ID, p, nil) {
		return true
	}
	return false
}

func (c RBACConfig) IsAllowedMany(user User, ps []gorbac.Permission) bool {
	for _, requiredPerm := range ps {
		// if none of the user's roles allow the state perm return false
		if !c.IsAllowed(user, requiredPerm) {
			return false
		}
	}
	return true
}

func (c RBACConfig) IsAllowedToExecute(user User, cmd Command) bool {
	allowed := true
	for _, p := range cmd.Permissions {
		if !c.IsAllowed(user, p) {
			allowed = false
			break
		}
	}
	return allowed
}

func (c RBACConfig) IsAllowedToExecuteMany(user User, cmds []Command) bool {
	allowed := true
	for _, cmd := range cmds {
		if !c.IsAllowedToExecute(user, cmd) {
			allowed = false
			break
		}
	}
	return allowed
}
