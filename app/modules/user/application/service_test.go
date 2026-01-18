package userservice

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	results "github.com/Black-And-White-Club/frolf-bot-shared/utils/results"

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

				mockRepo := userdb.NewMockRepository(ctrl)
				mockMetrics := mocks.NewMockUserMetrics(ctrl)
				tracer := noop.NewTracerProvider().Tracer("test")

				service := NewUserService(mockRepo, logger, mockMetrics, tracer)

				if service == nil {
					t.Fatalf("NewUserService returned nil")
				}

				// We can override behavior by calling withTelemetry directly via a no-op wrapper function in tests if needed.

				if service.repo != mockRepo {
					t.Error("repo not set correctly")
				}

				if service.logger != logger {
					t.Errorf("logger not correctly assigned")
				}
				if service.metrics != mockMetrics {
					t.Errorf("metrics not correctly assigned")
				}
				if service.tracer != tracer {
					t.Errorf("tracer not correctly assigned")
				}
			},
		},
		{
			name: "Handles nil dependencies",
			test: func(t *testing.T) {
				service := NewUserService(nil, nil, nil, nil)

				if service == nil {
					t.Fatal("NewGuildService returned nil")
				}

				if service.repo != nil {
					t.Error("repo should be nil")
				}

				if service.logger != nil {
					t.Error("logger should be nil")
				}
				if service.metrics != nil {
					t.Error("metrics should be nil")
				}
				if service.tracer != nil {
					t.Error("tracer should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestUserService_WithTelemetry(t *testing.T) {
	type args struct {
		ctx           context.Context
		operationName string
		userID        sharedtypes.DiscordID
		op            func(ctx context.Context) (results.OperationResult, error)
		serviceFunc   func(ctx context.Context) (results.OperationResult, error)
		logger        *slog.Logger
		metrics       usermetrics.UserMetrics
		tracer        trace.Tracer
	}
	tests := []struct {
		name    string
		args    func(ctrl *gomock.Controller) args
		want    results.OperationResult
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
					serviceFunc: func(ctx context.Context) (results.OperationResult, error) {
						return results.SuccessResult("test"), nil
					},
					logger:  logger,
					metrics: mockMetrics,
					tracer:  tracer,
				}
			},
			want:    results.OperationResult{Success: "test"},
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
					serviceFunc: func(ctx context.Context) (results.OperationResult, error) {
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
			name: "Handles operation returning an error",
			args: func(ctrl *gomock.Controller) args {
				testHandler := loggerfrolfbot.NewTestHandler()
				logger := slog.New(testHandler)
				metrics := &usermetrics.NoOpMetrics{}
				tracer := noop.NewTracerProvider().Tracer("test")

				return args{
					ctx:           context.Background(),
					operationName: "TestOperation",
					userID:        sharedtypes.DiscordID("123"),
					op: func(ctx context.Context) (results.OperationResult, error) {
						return results.OperationResult{}, errors.New("service error")
					},
					logger:  logger,
					metrics: metrics,
					tracer:  tracer,
				}
			},
			wantErr: true,
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

			// Build a temporary UserService to exercise withTelemetry
			s := &UserService{
				logger:  testArgs.logger,
				metrics: testArgs.metrics,
				tracer:  testArgs.tracer,
			}
			got, err := s.withTelemetry(testArgs.ctx, testArgs.operationName, testArgs.userID, testArgs.serviceFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("withTelemetry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Success != tt.want.Success {
				t.Errorf("withTelemetry() Success = %v, want %v", got.Success, tt.want.Success)
			}
			if got.Failure != tt.want.Failure {
				t.Errorf("withTelemetry() Failure = %v, want %v", got.Failure, tt.want.Failure)
			}
		})
	}
}

func TestUserService_MatchParsedScorecard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockRepository(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testRoundID := sharedtypes.RoundID(uuid.New())
	testImportID := "import-123"
	testUserID := sharedtypes.DiscordID("user-123")

	tests := []struct {
		name      string
		payload   roundevents.ParsedScorecardPayloadV1
		mockSetup func()
		want      results.OperationResult
		wantErr   bool
	}{
		{
			name: "Successful match by username",
			payload: roundevents.ParsedScorecardPayloadV1{
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
					Return(&userrepo.UserWithMembership{
						User: &userrepo.User{UserID: "discord-user-1"},
						Role: sharedtypes.UserRoleUser,
					}, nil)
			},
			want: results.OperationResult{
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
			payload: roundevents.ParsedScorecardPayloadV1{
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
					Return(nil, userrepo.ErrNotFound)
				mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, "test user").
					Return(&userrepo.UserWithMembership{
						User: &userrepo.User{UserID: "discord-user-2"},
						Role: sharedtypes.UserRoleUser,
					}, nil)
			},
			want: results.OperationResult{
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
			payload: roundevents.ParsedScorecardPayloadV1{
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
					Return(nil, userrepo.ErrNotFound)
				mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, "unknown").
					Return(nil, userrepo.ErrNotFound)
			},
			want: results.OperationResult{
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
			payload: roundevents.ParsedScorecardPayloadV1{
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
			want:    results.OperationResult{Failure: errors.New("parsed_data is nil")},
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

func TestUserService_UpdateUDiscIdentity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockRepository(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testUserID := sharedtypes.DiscordID("user-123")
	username := "TestUser"
	name := "Test Name"

	t.Run("Successful update", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "UpdateUDiscIdentity", testUserID)
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "UpdateUDiscIdentity", gomock.Any(), testUserID)
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "UpdateUDiscIdentity", testUserID)

		// Expect normalized values - now calls UpdateUDiscIdentityGlobal (no guildID)
		mockDB.EXPECT().UpdateUDiscIdentityGlobal(gomock.Any(), testUserID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, uid sharedtypes.DiscordID, u *string, n *string) error {
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

		mockDB.EXPECT().UpdateUDiscIdentityGlobal(gomock.Any(), testUserID, gomock.Any(), gomock.Any()).
			Return(errors.New("db error"))

		_, err := service.UpdateUDiscIdentity(context.Background(), testGuildID, testUserID, &username, &name)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestUserService_FindByUDiscUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockRepository(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testUsername := "testuser"

	t.Run("User found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscUsername", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(&userrepo.UserWithMembership{
				User: &userrepo.User{UserID: "found-user"},
				Role: sharedtypes.UserRoleUser,
			}, nil)

		got, err := service.FindByUDiscUsername(context.Background(), testGuildID, testUsername)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Success.(*userrepo.UserWithMembership).User.UserID != "found-user" {
			t.Errorf("expected user ID found-user")
		}
	})

	t.Run("User not found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscUsername", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "FindByUDiscUsername", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscUsername(gomock.Any(), testGuildID, testUsername).
			Return(nil, userrepo.ErrNotFound)

		_, err := service.FindByUDiscUsername(context.Background(), testGuildID, testUsername)
		if !errors.Is(err, userrepo.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestUserService_FindByUDiscName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := userdb.NewMockRepository(ctrl)
	mockMetrics := mocks.NewMockUserMetrics(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracer := noop.NewTracerProvider().Tracer("test")

	service := NewUserService(mockDB, logger, mockMetrics, tracer)

	testGuildID := sharedtypes.GuildID("guild-123")
	testName := "test name"

	t.Run("User found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscName", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationSuccess(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testName).
			Return(&userrepo.UserWithMembership{
				User: &userrepo.User{UserID: "found-user"},
				Role: sharedtypes.UserRoleUser,
			}, nil)

		got, err := service.FindByUDiscName(context.Background(), testGuildID, testName)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got.Success.(*userrepo.UserWithMembership).User.UserID != "found-user" {
			t.Errorf("expected user ID found-user")
		}
	})

	t.Run("User not found", func(t *testing.T) {
		mockMetrics.EXPECT().RecordOperationAttempt(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationDuration(gomock.Any(), "FindByUDiscName", gomock.Any(), sharedtypes.DiscordID(""))
		mockMetrics.EXPECT().RecordOperationFailure(gomock.Any(), "FindByUDiscName", sharedtypes.DiscordID(""))

		mockDB.EXPECT().FindByUDiscName(gomock.Any(), testGuildID, testName).
			Return(nil, userrepo.ErrNotFound)

		_, err := service.FindByUDiscName(context.Background(), testGuildID, testName)
		if !errors.Is(err, userrepo.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
