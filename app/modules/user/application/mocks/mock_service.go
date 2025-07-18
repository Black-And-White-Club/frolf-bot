// Code generated by MockGen. DO NOT EDIT.
// Source: ./app/modules/user/application/interface.go
//
// Generated by this command:
//
//	mockgen -source=./app/modules/user/application/interface.go -destination=./app/modules/user/application/mocks/mock_service.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	gomock "go.uber.org/mock/gomock"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
	isgomock struct{}
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CreateUser mocks base method.
func (m *MockService) CreateUser(ctx context.Context, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber) (userservice.UserOperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, userID, tag)
	ret0, _ := ret[0].(userservice.UserOperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockServiceMockRecorder) CreateUser(ctx, userID, tag any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockService)(nil).CreateUser), ctx, userID, tag)
}

// GetUser mocks base method.
func (m *MockService) GetUser(ctx context.Context, userID sharedtypes.DiscordID) (userservice.UserOperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, userID)
	ret0, _ := ret[0].(userservice.UserOperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockServiceMockRecorder) GetUser(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockService)(nil).GetUser), ctx, userID)
}

// GetUserRole mocks base method.
func (m *MockService) GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (userservice.UserOperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserRole", ctx, userID)
	ret0, _ := ret[0].(userservice.UserOperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserRole indicates an expected call of GetUserRole.
func (mr *MockServiceMockRecorder) GetUserRole(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserRole", reflect.TypeOf((*MockService)(nil).GetUserRole), ctx, userID)
}

// UpdateUserRoleInDatabase mocks base method.
func (m *MockService) UpdateUserRoleInDatabase(ctx context.Context, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (userservice.UserOperationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserRoleInDatabase", ctx, userID, newRole)
	ret0, _ := ret[0].(userservice.UserOperationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUserRoleInDatabase indicates an expected call of UpdateUserRoleInDatabase.
func (mr *MockServiceMockRecorder) UpdateUserRoleInDatabase(ctx, userID, newRole any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserRoleInDatabase", reflect.TypeOf((*MockService)(nil).UpdateUserRoleInDatabase), ctx, userID, newRole)
}
