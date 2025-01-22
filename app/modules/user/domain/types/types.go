package usertypes

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// DiscordID defines a custom type for Discord IDs.
type DiscordID string

var discordIDRegex = regexp.MustCompile(`^[0-9]+$`) // Matches one or more digits

// IsValid checks if the DiscordID is valid (contains only numbers).
func (id DiscordID) IsValid() bool {
	return discordIDRegex.MatchString(string(id))
}

// User interface
type User interface {
	GetID() int64
	GetName() string
	GetDiscordID() DiscordID
	GetRole() UserRoleEnum // Change this to UserRoleEnum
}

// UserData struct implementing the User interface
type UserData struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	DiscordID DiscordID    `json:"discord_id"`
	Role      UserRoleEnum `json:"role"` // Use UserRoleEnum here
}

func (u UserData) GetID() int64 {
	return u.ID
}

func (u UserData) GetName() string {
	return u.Name
}

func (u UserData) GetDiscordID() DiscordID {
	return u.DiscordID
}

func (u UserData) GetRole() UserRoleEnum {
	return u.Role
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
func (ur *UserRoleEnum) UnmarshalJSONRole(data []byte) error {
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

// UnmarshalJSON implements custom unmarshaling for DiscordID from JSON.
func (id *DiscordID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if !discordIDRegex.MatchString(s) {
		return fmt.Errorf("invalid Discord ID format: %s", s)
	}

	*id = DiscordID(s)
	return nil
}

// ParseUserRoleEnum converts a string to a UserRoleEnum.
func ParseUserRoleEnum(role string) (UserRoleEnum, error) {
	switch role {
	case "Rattler":
		return UserRoleRattler, nil
	case "Editor":
		return UserRoleEditor, nil
	case "Admin":
		return UserRoleAdmin, nil
	default:
		return UserRoleUnknown, fmt.Errorf("invalid user role: %s", role)
	}
}
