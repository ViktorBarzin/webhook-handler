// Package auth is responsible for mapping users to their permissions within the bot
package auth

import (
	"github.com/viktorbarzin/gorbac"
)

const (
	GuestUserID = "__guest"
)

type Subject interface {
	Can(command Command) bool
}

type RBACConfig struct {
	Groups      []Group                `yaml:"groups"`
	Users       []User                 `yaml:"users"`
	Roles       []Role                 `yaml:"roles"`
	Permissions []gorbac.StdPermission `yaml:"permissions"`
	Commands    []Command              `yaml:"commands"`
	RBAC        *gorbac.RBAC
}

// Command is a shell cmd that can be executed by the chatbot
type Command struct {
	ID          string                 `yaml:"id"`
	PrettyName  string                 `yaml:"prettyName"`
	CMD         string                 `yaml:"cmd"`
	Permissions []gorbac.StdPermission `yaml:"permissions"`
}

// Role on the RBAC e.g "admin"
type Role struct {
	Name        string                 `yaml:"id"` // must be unique
	Permissions []gorbac.StdPermission `yaml:"permissions"`
}

// User is a kind of subject which represents a physical user
type User struct {
	ID     string  `yaml:"id"`
	Name   string  `yaml:"name"`
	Roles  []Role  `yaml:"roles"`
	Groups []Group `yaml:"groups"`
}

// GuestUser returns a guest user which is a handler for all non-present accounts
func GuestUser() User {
	return User{ID: GuestUserID, Name: "Guest"}
}

// Group is a kind if subject which corresponds to a set of users
type Group struct {
	Name  string `yaml:"name"`
	Roles []Role `yaml:"roles"`
}
