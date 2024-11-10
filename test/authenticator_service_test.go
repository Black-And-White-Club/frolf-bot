// authenticator_service.go
package services

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
)

func TestMiddleware(t *testing.T) {
	type args struct {
		firestoreClient *firestore.Client
	}
	tests := []struct {
		name string
		args args
		want func(http.Handler) http.Handler
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Middleware(tt.args.firestoreClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Middleware() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestForContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *User
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ForContext(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ForContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateAndGetUserID(t *testing.T) {
	type args struct {
		c *http.Cookie
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAndGetUserID(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAndGetUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validateAndGetUserID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUserByID(t *testing.T) {
	type args struct {
		client *firestore.Client
		ctx    context.Context
		userID string
	}
	tests := []struct {
		name    string
		args    args
		want    *User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getUserByID(tt.args.client, tt.args.ctx, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		u    *User
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsAdmin(); got != tt.want {
				t.Errorf("User.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsEditor(t *testing.T) {
	tests := []struct {
		name string
		u    *User
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsEditor(); got != tt.want {
				t.Errorf("User.IsEditor() = %v, want %v", got, tt.want)
			}
		})
	}
}
