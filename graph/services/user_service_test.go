package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/romero-jace/tcr-bot/graph/model"
)

func TestNewUserService(t *testing.T) {
	mockClient := &MockFirestoreClient{users: make(map[string]*model.User)} // Use your mock client

	tests := []struct {
		name string
		args FirestoreClient // Change to use the interface
		want *UserService
	}{
		{
			name: "Create UserService",
			args: mockClient, // Use the mock client
			want: &UserService{client: mockClient},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUserService(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUser Service() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	mockClient := &MockFirestoreClient{users: make(map[string]*model.User)}
	us := &UserService{client: mockClient}

	tests := []struct {
		name    string
		args    model.UserInput
		want    *model.User
		wantErr bool
	}{
		{
			name:    "Create User Success",
			args:    model.UserInput{DiscordID: "12345", Name: "Test User"},
			want:    &model.User{DiscordID: "12345", Name: "Test User"},
			wantErr: false,
		},
		{
			name:    "Create User Failure - Missing DiscordID",
			args:    model.UserInput{Name: "Test User"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Create User Failure - Missing Name",
			args:    model.UserInput{DiscordID: "12345"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Create User Failure - User Already Exists",
			args:    model.UserInput{DiscordID: "12345", Name: "Test User"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Create User Failure - User Already Exists" {
				// Pre-create the user to simulate existing user
				mockClient.users[tt.args.DiscordID] = &model.User{DiscordID: tt.args.DiscordID, Name: tt.args.Name}
			}
			got, err := us.CreateUser(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("User Service.CreateUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("User Service.CreateUser () = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	mockClient := &MockFirestoreClient{users: make(map[string]*model.User)}
	us := &UserService{client: mockClient}

	tests := []struct {
		name    string
		args    string
		want    *model.User
		wantErr bool
	}{
		{
			name:    "Get User Success",
			args:    "12345",
			want:    &model.User{DiscordID: "12345", Name: "Test User"},
			wantErr: false,
		},
		{
			name:    "Get User Failure - User Not Found",
			args:    "nonexistent",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Get User Failure - Missing DiscordID",
			args:    "",
			want:    nil,
			wantErr: true,
		},
	}

	// Pre-create a user for the success case
	mockClient.users["12345"] = &model.User{DiscordID: "12345", Name: "Test User"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := us.GetUser(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("User  Service.GetUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("User  Service.GetUser () = %v, want %v", got, tt.want)
			}
		})
	}
}
