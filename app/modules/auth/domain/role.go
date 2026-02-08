package authdomain

// Role represents a user's role for authorization purposes.
type Role string

const (
	RoleViewer Role = "viewer"
	RolePlayer Role = "player"
	RoleEditor Role = "editor"
	RoleAdmin  Role = "admin"
)

// IsValid checks if the role is a valid value.
func (r Role) IsValid() bool {
	switch r {
	case RoleViewer, RolePlayer, RoleEditor, RoleAdmin:
		return true
	default:
		return false
	}
}

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}
