// Package auth is responsible for mapping users to their permissions within the bot
package auth

type RBACConfig struct {
	Groups      []Group      `yaml:"groups"`
	Users       []User       `yaml:"users"`
	Roles       []Role       `yaml:"roles"`
	Permissions []Permission `yaml:"permissions"`
	Commands    []Command    `yaml:"command"`
}

// Permission is a permission in the cluster e.g "execute"
type Permission struct {
	ID       string    `yaml:"name"`
	Commands []Command `yaml:"commands"`
}

// Command is a shell cmd that can be executed by the chatbot
type Command struct {
	ID         string `yaml:"id"`
	PrettyName string `yaml:"prettyName"`
	CMD        string `yaml:"cmd"`
}

// Role on the RBAC e.g "admin"
type Role struct {
	Name        string       `yaml:"name"` // must be unique
	Permissions []Permission `yaml:"permissions"`
}

// User is a kind of subject which represents a physical user
type User struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Group is a kind if subject which corresponds to a set of users
type Group struct {
	Name    string `yaml:"name"`
	Members []User `yaml:"members"`
	Roles   []Role `yaml:"roles"`
}
