package usertypes

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// User defines the methods for accessing user data.
type User interface {
	GetID() int64
	GetName() string
	GetDiscordID() DiscordID
	GetRole() UserRoleEnum // Change this to UserRoleEnum
}

// DiscordID defines a custom type for Discord IDs.
type DiscordID string

var discordIDRegex = regexp.MustCompile(`^[0-9]+$`) // Matches one or more digits

// IsValid checks if the DiscordID is valid (contains only numbers).
func (id DiscordID) IsValid() bool {
	return discordIDRegex.MatchString(string(id))
}

// UserRoleEnum represents the role of a user.
type UserRoleEnum string

// Constants for user roles
const (
	UserRoleUnknown UserRoleEnum = ""
	UserRoleRattler UserRoleEnum = "Rattler"
	UserRoleEditor  UserRoleEnum = "Editor"
	UserRoleAdmin   UserRoleEnum = "Admin"
)

// IsValid checks if the given role is valid.
func (ur UserRoleEnum) IsValid() bool {
	switch ur {
	case UserRoleRattler, UserRoleEditor, UserRoleAdmin:
		return true
	default:
		return false
	}
}

// String returns the string representation of the UserRoleEnum.
func (ur UserRoleEnum) String() string {
	return string(ur)
}

// MarshalJSON marshals the UserRoleEnum to JSON.
func (ur UserRoleEnum) MarshalJSON() ([]byte, error) {
	return json.Marshal(ur.String())
}

// UnmarshalJSON unmarshals the UserRoleEnum from JSON.
func (ur *UserRoleEnum) UnmarshalJSON(data []byte) error {
	var roleStr string
	if err := json.Unmarshal(data, &roleStr); err != nil {
		return err
	}

	switch roleStr {
	case "Rattler":
		*ur = UserRoleRattler
	case "Editor":
		*ur = UserRoleEditor
	case "Admin":
		*ur = UserRoleAdmin
	default:
		return fmt.Errorf("invalid UserRoleEnum: %s", roleStr)
	}

	return nil
}
