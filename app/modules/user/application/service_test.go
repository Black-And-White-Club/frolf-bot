package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userrepo "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Creates service with all dependencies",
			test: func(t *testing.T) {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockDB := userdb.NewMockUserDB(ctrl)
				mockEventBus := eventbus.NewMockEventBus(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				// Override serviceWrapper to avoid side effects during test
				userServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				}

				if userServiceImpl.UserDB != mockDB {
					t.Errorf("UserDB not correctly assigned")
				}
				if userServiceImpl.eventBus != mockEventBus {
					t.Errorf("eventBus not correctly assigned")
				}
				if userServiceImpl.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if userServiceImpl.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if userServiceImpl.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}

				if userServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				service := NewUserService(nil, nil, nil, nil, nil)

				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				userServiceImpl, ok := service.(*UserServiceImpl)
				if !ok {
					t.Fatalf("service is not of type *UserServiceImpl")
				}

				userServiceImpl.serviceWrapper = func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
					return serviceFunc(ctx)
				}

				if userServiceImpl.UserDB != nil {
					t.Errorf("UserDB should be nil")
				}
				if userServiceImpl.eventBus != nil {
					t.Errorf("eventBus should be nil")
				}
				if userServiceImpl.logger != nil {
					t.Errorf("logger should be nil")
				}
				if userServiceImpl.metrics != nil {
					t.Errorf("metrics should be nil")
				}
				if userServiceImpl.tracer != nil {
					t.Errorf("tracer should be nil")
				}

				if userServiceImpl.serviceWrapper == nil {
					t.Errorf("serviceWrapper should not be nil")
				}

				ctx := context.Background()
				_, err := userServiceImpl.serviceWrapper(ctx, "TestOp", "123", func(ctx context.Context) (UserOperationResult, error) {
					return UserOperationResult{Success: "test"}, nil
				})
				if err != nil {
					t.Errorf("serviceWrapper should execute the provided function without error, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func Test_serviceWrapper(t *testing.T) {
	type args struct {
		ctx           context.Context
		operationName string
		userID        sharedtypes.DiscordID
		serviceFunc   func(ctx context.Context) (UserOperationResult, error)
		logger        *slog.Logger
		metrics       usermetrics.UserMetrics
		tracer        trace.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    UserOperationResult
		wantErr bool
		setup   func(a *args, ctx context.Context)
	}{
		{
			name: "Successful operation",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						return UserOperationResult{Success: "test"}, nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			want:    UserOperationResult{Success: "test"},
			wantErr: false,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Info("Operation succeeded", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationSuccess(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
		{
			name: "Handles panic in service function",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						panic("test panic")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Error("Panic in TestOperation: test panic", attr.String("user_id", "123"), attr.Any("panic", "test panic"))
				mockMetrics.EXPECT().RecordOperationFailure(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
		{
			name: "Handles service function returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					serviceFunc: func(ctx context.Context) (UserOperationResult, error) {
						return UserOperationResult{}, fmt.Errorf("service error")
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
			setup: func(a *args, ctx context.Context) {
				mockMetrics := a.metrics.(*mocks.MockUserMetrics)
				mockLogger := a.logger

				mockMetrics.EXPECT().RecordOperationAttempt(ctx, "TestOperation", sharedtypes.DiscordID("123"))
				mockLogger.Info("Starting operation", attr.String("operation", "TestOperation"), attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationDuration(ctx, "TestOperation", gomock.Any(), gomock.Any())
				mockLogger.Error("Error in TestOperation: service error", attr.String("user_id", "123"))
				mockMetrics.EXPECT().RecordOperationFailure(ctx, "TestOperation", sharedtypes.DiscordID("123"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			testArgs := tt.args(ctrl)
			ctx, span := testArgs.tracer.Start(testArgs.ctx, testArgs.operationName)
			defer span.End()

			if tt.setup != nil {
				tt.setup(&testArgs, ctx)
			}

			got, err := serviceWrapper(testArgs.ctx, testArgs.operationName, testArgs.userID, testArgs.serviceFunc, testArgs.logger, testArgs.metrics, testArgs.tracer)
			if (err != nil) != tt.wantErr {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Success != tt.want.Success {
				t.Errorf("serviceWrapper() Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Failure != tt.want.Failure {
				t.Errorf("serviceWrapper() Failure = %v, want %v", got.Failure, tt.want.Failure)
			}
		})
	}
}

func TestUserServiceImpl_MatchParsedScorecard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testImportID := "import-123"
	testUserID := sharedtypes.DiscordID("user-123")

	tests := []struct {
		name      string
		payload   roundevents.ParsedScorecardPayload
		mockSetup func()
		want      UserOperationResult
		wantErr   bool
	}{
		{
			name: "Successful match by username",
			payload: roundevents.ParsedScorecardPayload{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
				UserID:   testUserID,
				ParsedData: &roundtypes.ParsedScorecard{
					PlayerScores: []roundtypes.PlayerScoreRow{
						{PlayerName: "TestUser"},
					},
				},
			},
			mockSetup: func() {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "MatchParsedScorecard", testUserID)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "MatchParsedScorecard", gomock.Any(), testUserID)
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "MatchParsedScorecard", testUserID)

				mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, "testuser").
					Return(&userrepo.User{UserID: "discord-user-1"}, nil)
			},
			want: UserOperationResult{
				Success: &userevents.UDiscMatchConfirmedPayloadV1{
					ImportID: testImportID,
					GuildID:  testGuildID,
					RoundID:  testRoundID,
					UserID:   testUserID,
					Mappings: []userevents.UDiscConfirmedMappingV1{
						{PlayerName: "TestUser", DiscordUserID: "discord-user-1"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Successful match by name (fallback)",
			payload: roundevents.ParsedScorecardPayload{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
				UserID:   testUserID,
				ParsedData: &roundtypes.ParsedScorecard{
					PlayerScores: []roundtypes.PlayerScoreRow{
						{PlayerName: "Test User"},
					},
				},
			},
			mockSetup: func() {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "MatchParsedScorecard", testUserID)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "MatchParsedScorecard", gomock.Any(), testUserID)
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "MatchParsedScorecard", testUserID)

				mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, "test user").
					Return(nil, userrepo.ErrUserNotFound)
				mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, "test user").
					Return(&userrepo.User{UserID: "discord-user-2"}, nil)
			},
			want: UserOperationResult{
				Success: &userevents.UDiscMatchConfirmedPayloadV1{
					ImportID: testImportID,
					GuildID:  testGuildID,
					RoundID:  testRoundID,
					UserID:   testUserID,
					Mappings: []userevents.UDiscConfirmedMappingV1{
						{PlayerName: "Test User", DiscordUserID: "discord-user-2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "No match found (skipped)",
			payload: roundevents.ParsedScorecardPayload{
				ImportID: testImportID,
				GuildID:  testGuildID,
				RoundID:  testRoundID,
				UserID:   testUserID,
				ParsedData: &roundtypes.ParsedScorecard{
					PlayerScores: []roundtypes.PlayerScoreRow{
						{PlayerName: "Unknown"},
					},
				},
			},
			mockSetup: func() {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "MatchParsedScorecard", testUserID)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "MatchParsedScorecard", gomock.Any(), testUserID)
				mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "MatchParsedScorecard", testUserID)

				mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, "unknown").
					Return(nil, userrepo.ErrUserNotFound)
				mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, "unknown").
					Return(nil, userrepo.ErrUserNotFound)
			},
			want: UserOperationResult{
				Success: &userevents.UDiscMatchConfirmedPayloadV1{
					ImportID: testImportID,
					GuildID:  testGuildID,
					RoundID:  testRoundID,
					UserID:   testUserID,
					Mappings: nil, // Empty mappings
				},
			},
			wantErr: false,
		},
		{
			name: "Parsed data is nil",
			payload: roundevents.ParsedScorecardPayload{
				ImportID:   testImportID,
				GuildID:    testGuildID,
				RoundID:    testRoundID,
				UserID:     testUserID,
				ParsedData: nil,
			},
			mockSetup: func() {
				mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "MatchParsedScorecard", testUserID)
				mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "MatchParsedScorecard", gomock.Any(), testUserID)
				mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "MatchParsedScorecard", testUserID)
			},
			want:    UserOperationResult{Failure: errors.New("parsed_data is nil")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			got, err := service.MatchParsedScorecard(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchParsedScorecard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify success payload
				gotPayload, ok := got.Success.(*userevents.UDiscMatchConfirmedPayloadV1)
				if !ok {
					t.Errorf("MatchParsedScorecard() success type mismatch, got %T", got.Success)
					return
				}
				wantPayload := tt.want.Success.(*userevents.UDiscMatchConfirmedPayloadV1)

				if gotPayload.ImportID != wantPayload.ImportID {
					t.Errorf("ImportID mismatch")
				}
				if len(gotPayload.Mappings) != len(wantPayload.Mappings) {
					t.Errorf("Mappings length mismatch")
				}
			}
		})
	}
}

func TestUserServiceImpl_UpdateUDiscIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("user-123")
	username := "TestUser"
	name := "Test Name"

	t.Run("Successful update", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "UpdateUDiscIdentity", testUserID)
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "UpdateUDiscIdentity", gomock.Any(), testUserID)
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "UpdateUDiscIdentity", testUserID)

		// Expect normalized values
		mockDB.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, gid sharedtypes.GuildID, uid sharedtypes.DiscordID, u *string, n *string) error {
				if *u != "testuser" {
					t.Errorf("expected normalized username 'testuser', got '%s'", *u)
				}
				if *n != "test name" {
					t.Errorf("expected normalized name 'test name', got '%s'", *n)
				}
				return nil
			})

		got, err := service.UpdateUDiscIdentity(context.Background(), testGuildID, testUserID, &username, &name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Success != true {
			t.Errorf("expected success true")
		}
	})

	t.Run("DB error", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "UpdateUDiscIdentity", testUserID)
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "UpdateUDiscIdentity", gomock.Any(), testUserID)
		mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "UpdateUDiscIdentity", testUserID)

		mockDB.EXPECT().UpdateUDiscIdentity(gomock.Any(), testGuildID, testUserID, gomock.Any(), gomock.Any()).
			Return(errors.New("db error"))

		_, err := service.UpdateUDiscIdentity(context.Background(), testGuildID, testUserID, &username, &name)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestUserServiceImpl_FindByUDiscUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testUsername := "testuser"

	t.Run("User found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscUsername", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(&userrepo.User{UserID: "found-user"}, nil)

		got, err := service.FindByUDiscUsername(context.Background(), testGuildID, testUsername)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Success.(*userrepo.User).UserID != "found-user" {
			t.Errorf("expected user ID found-user")
		}
	})

	t.Run("User not found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscUsername", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(nil, userrepo.ErrUserNotFound)

		_, err := service.FindByUDiscUsername(context.Background(), testGuildID, testUsername)
		if !errors.Is(err, userrepo.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestUserServiceImpl_FindByUDiscName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockUserDB(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, mockEventBus, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testName := "test name"

	t.Run("User found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscName", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testName).
			Return(&userrepo.User{UserID: "found-user"}, nil)

		got, err := service.FindByUDiscName(context.Background(), testGuildID, testName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Success.(*userrepo.User).UserID != "found-user" {
			t.Errorf("expected user ID found-user")
		}
	})

	t.Run("User not found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscName", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testName).
			Return(nil, userrepo.ErrUserNotFound)

		_, err := service.FindByUDiscName(context.Background(), testGuildID, testName)
		if !errors.Is(err, userrepo.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
