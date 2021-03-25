// Package auth is responsible for mapping users to their permissions within the bot
package auth

import (
	"github.com/viktorbarzin/gorbac"
)

const (
	GuestUserID = "__guest"
)

type RBACConfig struct {
	Groups      []Group                `yaml:"groups" json:"groups"`
	Users       []User                 `yaml:"users" json:"users"`
	Roles       []Role                 `yaml:"roles" json:"roles"`
	Permissions []gorbac.StdPermission `yaml:"permissions" json:"permissions"`
	Commands    []Command              `yaml:"commands" json:"commands"`
	RBAC        *gorbac.RBAC
}

// Command is a shell cmd that can be executed by the chatbot
type Command struct {
	ID          string                 `yaml:"id" json:"id"`
	PrettyName  string                 `yaml:"prettyName" json:"prettyName"`
	CMD         string                 `yaml:"cmd" json:"cmd"`
	Permissions []gorbac.StdPermission `yaml:"permissions" json:"permissions"`
	// Message to send the user after the command succeeded
	SuccessExplanation string `yaml:"onSuccess"`
	ShowCmdOutput      bool   `yaml:"showCmdOutput" json:"showCmdOutput"`
	ApprovedBy         Role   `yaml:"approvedBy" json:"approvedBy"`
}

// Role on the RBAC e.g "admin"
type Role struct {
	Name        string                 `yaml:"id" json:"id"` // must be unique
	Permissions []gorbac.StdPermission `yaml:"permissions" json:"permissions"`
}

// User is a kind of subject which represents a physical user
type User struct {
	ID     string  `yaml:"id" json:"id"`
	Name   string  `yaml:"name" json:"name"`
	Roles  []Role  `yaml:"roles" json:"roles"`
	Groups []Group `yaml:"groups" json:"groups"`
}

// GuestUser returns a guest user which is a handler for all non-present accounts
func GuestUser() User {
	return GuestUserWithId(GuestUserID)
}

func GuestUserWithId(id string) User {
	return User{ID: id, Name: "Guest"}
}

// Group is a kind if subject which corresponds to a set of users
type Group struct {
	Name  string `yaml:"name" json:"name"`
	Roles []Role `yaml:"roles" json:"roles"`
}
