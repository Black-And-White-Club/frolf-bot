package userservice

import (
	"context"
	"errors"
	"testing"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	mockdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestUserServiceImpl_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	testUserID := sharedtypes.DiscordID("12345678901234567")
	testGuildID := sharedtypes.GuildID("98765432109876543")
	testTag := sharedtypes.TagNumber(42)

	logger := loggerfrolfbot.NoOpLogger
	metrics := &usermetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	tests := []struct {
		name               string
		mockDBSetup        func(*mockdb.MockUserDB)
		userID             sharedtypes.DiscordID
		guildID            sharedtypes.GuildID
		tag                *sharedtypes.TagNumber
		expectedSuccess    bool
		expectedIsReturning bool
		expectedReason     string
		expectErr          bool
	}{
		{
			name: "New user creation (IsReturningUser=false)",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// Step 1: User doesn't exist globally
				m.EXPECT().GetUserGlobal(gomock.Any(), testUserID).Return(nil, errors.New("not found"))
				// Step 2: Create global user succeeds
				m.EXPECT().CreateGlobalUser(gomock.Any(), gomock.Any()).Return(nil)
				// Step 3: Create guild membership succeeds
				m.EXPECT().CreateGuildMembership(gomock.Any(), gomock.Any()).Return(nil)
			},
			userID:              testUserID,
			guildID:             testGuildID,
			tag:                 &testTag,
			expectedSuccess:     true,
			expectedIsReturning: false,
			expectErr:           false,
		},
		{
			name: "Returning user to new guild (IsReturningUser=true)",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// Step 1: User exists globally
				existingUser := &userdb.User{UserID: testUserID}
				m.EXPECT().GetUserGlobal(gomock.Any(), testUserID).Return(existingUser, nil)
				// Step 1b: User not yet in this guild
				m.EXPECT().GetGuildMembership(gomock.Any(), testUserID, testGuildID).Return(nil, errors.New("not found"))
				// Step 3: Create guild membership succeeds
				m.EXPECT().CreateGuildMembership(gomock.Any(), gomock.Any()).Return(nil)
			},
			userID:              testUserID,
			guildID:             testGuildID,
			tag:                 nil,
			expectedSuccess:     true,
			expectedIsReturning: true,
			expectErr:           false,
		},
		{
			name: "User already exists in guild (failure)",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// Step 1: User exists globally
				existingUser := &userdb.User{UserID: testUserID}
				m.EXPECT().GetUserGlobal(gomock.Any(), testUserID).Return(existingUser, nil)
				// Step 1b: User already in this guild
				existingMembership := &userdb.GuildMembership{UserID: testUserID, GuildID: testGuildID}
				m.EXPECT().GetGuildMembership(gomock.Any(), testUserID, testGuildID).Return(existingMembership, nil)
			},
			userID:          testUserID,
			guildID:         testGuildID,
			tag:             &testTag,
			expectedSuccess: false,
			expectedReason:  "user already exists in this guild",
			expectErr:       true,
		},
		{
			name: "CreateGlobalUser fails (duplicate user error)",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// Step 1: User doesn't exist globally
				m.EXPECT().GetUserGlobal(gomock.Any(), testUserID).Return(nil, errors.New("not found"))
				// Step 2: CreateGlobalUser fails with duplicate error
				m.EXPECT().CreateGlobalUser(gomock.Any(), gomock.Any()).Return(errors.New("SQLSTATE 23505: duplicate key value"))
			},
			userID:          testUserID,
			guildID:         testGuildID,
			tag:             nil,
			expectedSuccess: false,
			expectedReason:  "user already exists",
			expectErr:       true,
		},
		{
			name: "CreateGuildMembership fails",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// Step 1: User doesn't exist globally
				m.EXPECT().GetUserGlobal(gomock.Any(), testUserID).Return(nil, errors.New("not found"))
				// Step 2: CreateGlobalUser succeeds
				m.EXPECT().CreateGlobalUser(gomock.Any(), gomock.Any()).Return(nil)
				// Step 3: CreateGuildMembership fails
				m.EXPECT().CreateGuildMembership(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
			},
			userID:          testUserID,
			guildID:         testGuildID,
			tag:             &testTag,
			expectedSuccess: false,
			expectedReason:  "database error",
			expectErr:       true,
		},
		{
			name: "Empty Discord ID validation",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// No DB calls expected
			},
			userID:          "",
			guildID:         testGuildID,
			tag:             &testTag,
			expectedSuccess: false,
			expectedReason:  "invalid Discord ID",
			expectErr:       true,
		},
		{
			name: "Negative tag number validation",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// No DB calls expected
			},
			userID:          testUserID,
			guildID:         testGuildID,
			tag:             func() *sharedtypes.TagNumber { t := sharedtypes.TagNumber(-1); return &t }(),
			expectedSuccess: false,
			expectedReason:  "tag number cannot be negative",
			expectErr:       true,
		},
		{
			name: "Nil context validation",
			mockDBSetup: func(m *mockdb.MockUserDB) {
				// No DB calls expected
			},
			userID:          testUserID,
			guildID:         testGuildID,
			tag:             &testTag,
			expectedSuccess: false,
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mockdb.NewMockUserDB(ctrl)
			tt.mockDBSetup(mockDB)

			s := &UserServiceImpl{
				UserDB:  mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			ctxArg := ctx
			if tt.name == "Nil context validation" {
				ctxArg = nil
			}

			result, err := s.CreateUser(ctxArg, tt.guildID, tt.userID, tt.tag, nil, nil)

			// Check success/failure
			if tt.expectedSuccess {
				if result.Success == nil {
					t.Errorf("Expected success, got failure: %v", result.Failure)
				} else {
					payload := result.Success.(*userevents.UserCreatedPayloadV1)
					if payload.UserID != tt.userID {
						t.Errorf("UserID mismatch: expected %q, got %q", tt.userID, payload.UserID)
					}
					if payload.IsReturningUser != tt.expectedIsReturning {
						t.Errorf("IsReturningUser mismatch: expected %v, got %v", tt.expectedIsReturning, payload.IsReturningUser)
					}
				}
			} else {
				// For nil context validation, expect error directly in result.Error
				if tt.name == "Nil context validation" {
					if result.Error == nil {
						t.Errorf("Expected error in result, got nil")
					}
				} else if result.Failure == nil {
					t.Errorf("Expected failure, got success")
				} else {
					failurePayload := result.Failure.(*userevents.UserCreationFailedPayloadV1)
					if tt.expectedReason != "" && failurePayload.Reason != tt.expectedReason {
						t.Errorf("Reason mismatch: expected %q, got %q", tt.expectedReason, failurePayload.Reason)
					}
				}
			}

			// Check error
			if tt.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
			} else if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}
