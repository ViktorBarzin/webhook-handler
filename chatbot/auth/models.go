// Package auth is responsible for mapping users to their permissions within the bot
// Users and Groups are kind of Subjects. Subjects are assigned Roles using RoleBindings. Roles have Permissions.
package auth

// Permission is a permission in the cluster e.g "execute"
type Permission struct {
	ID string
}

// Command is a shell cmd that can be executed by the chatbot
type Command struct {
	PrettyName string
	CMD        string
}

// PermissionCommandMapping maps permissions to respective commands
type PermissionCommandMapping struct {
	Permissions map[Permission][]Command
}

// Role on the RBAC e.g "admin"
type Role struct {
	Name        string // must be unique
	Permissions []Permission
}

// RoleBinding maps subjects to a list of roles they are bound to
type RoleBinding struct {
	Bindings map[Subject][]Role
}

// Subject is an interface which represents an entity which can have permissions
type Subject interface {
	GetID() string
}

// User is a kind of subject which represents a physical user
type User struct {
	ID   string
	Name string
}

// Group is a kind if subject which corresponds to a set of users
type Group struct {
	ID      string
	Name    string
	Members []User
}
