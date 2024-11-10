// authenticator_service.go
package services

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"
)

// A private key for context that only this package can access
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// User represents an authenticated user
type User struct {
	Name string
	Role string // Add a Role field to represent user roles
}

const (
	RoleUser   = "user"
	RoleEditor = "editor"
	RoleAdmin  = "admin"
)

// Middleware decodes the shared session cookie and packs the session into context
func Middleware(firestoreClient *firestore.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("auth-cookie")

			// Allow unauthenticated users in
			if err != nil || c == nil {
				next.ServeHTTP(w, r)
				return
			}

			userId, err := validateAndGetUserID(c)
			if err != nil {
				http.Error(w, "Invalid cookie", http.StatusForbidden)
				return
			}

			// Get the user from Firestore
			user, err := getUserByID(firestoreClient, userId)
			if err != nil {
				http.Error(w, "User  not found", http.StatusForbidden)
				return
			}

			// Put it in context
			ctx := context.WithValue(r.Context(), userCtxKey, user)

			// Call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *User {
	raw, _ := ctx.Value(userCtxKey).(*User)
	return raw
}

// validateAndGetUser ID validates the cookie and returns the user ID
func validateAndGetUserID(c *http.Cookie) (string, error) {
	// Implement your cookie validation logic here
	// If valid, return the user ID
	return "user-id", nil // Replace with actual logic
}

// getUser ByID retrieves the user from Firestore
func getUserByID(client *firestore.Client, userID string) (*User, error) {
	docRef := client.Collection("users").Doc(userID)
	docSnap, err := docRef.Get(context.Background())
	if err != nil {
		return nil, err
	}

	var user User
	err = docSnap.DataTo(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsEditor checks if the user is an editor
func (u *User) IsEditor() bool {
	return u.Role == RoleEditor || u.IsAdmin() // Editors can also be admins
}
