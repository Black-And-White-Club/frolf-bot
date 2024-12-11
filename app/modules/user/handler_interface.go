// user/handler_interface.go
package user

import (
	"net/http"
)

type UserHandler interface {
	CreateUser(w http.ResponseWriter, r *http.Request)
	GetUser(w http.ResponseWriter, r *http.Request)
	UpdateUser(w http.ResponseWriter, r *http.Request)
	GetUserRole(w http.ResponseWriter, r *http.Request)
}
