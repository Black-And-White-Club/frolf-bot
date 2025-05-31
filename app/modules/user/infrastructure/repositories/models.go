package userdb

import (
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// User represents a user in the system.
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	ID            int64                    `bun:"id,pk,autoincrement" json:"id"`
	UserID        sharedtypes.DiscordID    `bun:"user_id,unique"`
	Role          sharedtypes.UserRoleEnum `bun:"role,notnull,default:'Rattler'" json:"role"`
}

// Add these methods to your User struct
func (u *User) GetID() int64 {
	return u.ID
}

// func (u *User) GetName() string {
// 	return u.Name
// }

func (u *User) GetUserID() sharedtypes.DiscordID {
	return u.UserID
}

func (u *User) GetRole() sharedtypes.UserRoleEnum {
	return u.Role
}

// UserData struct implementing the User interface
type UserData struct {
	ID int64 `json:"id"`
	// Name      string                 `json:"name"`
	UserID sharedtypes.DiscordID    `json:"user_id"`
	Role   sharedtypes.UserRoleEnum `json:"role"` // Use UserRoleEnum here
}
