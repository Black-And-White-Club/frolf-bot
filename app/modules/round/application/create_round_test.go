package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func timePtr(t time.Time) *sharedtypes.StartTime {
	st := sharedtypes.StartTime(t)
	return &st
}

func TestRoundService_ValidateAndProcessRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockTimeParser := roundutil.NewMockTimeParserInterface(ctrl)
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockTimeParserSetup     func(*roundutil.MockTimeParserInterface)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		payload                 roundevents.CreateRoundRequestedPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "valid round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundutil.MockTimeParserInterface) {
				mockTimeParser.EXPECT().ParseUserTimeInput(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1884312000), nil) // 2029-09-16T12:00:00Z
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				mockRoundValidator.EXPECT().ValidateRoundInput(gomock.Any()).Return([]string{})
			},
			payload: roundevents.CreateRoundRequestedPayload{
				Title:       roundtypes.Title("Test Round"),
				Description: roundtypes.Description("Test Description"),
				StartTime:   "2029-09-16T12:00:00Z", // updated start time
				Location:    roundtypes.Location("Test Location"),
				UserID:      "Test User",
				ChannelID:   "Test Channel",
				Timezone:    roundtypes.Timezone("America/New_York"),
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundEntityCreatedPayload{
					Round: roundtypes.Round{
						Title:        roundtypes.Title("Test Round"),
						Description:  roundtypes.DescriptionPtr("Test Description"),
						Location:     roundtypes.LocationPtr("Test Location"),
						StartTime:    (*sharedtypes.StartTime)(timePtr(time.Unix(1884312000, 0))),
						CreatedBy:    sharedtypes.DiscordID("Test User"),
						State:        roundtypes.RoundStateUpcoming,
						Participants: []roundtypes.Participant{},
					},
					DiscordChannelID: "Test Channel",
					DiscordGuildID:   "",
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundutil.MockTimeParserInterface) {
				// mockTimeParser.EXPECT().ParseUserTimeInput(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1672531200), nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				mockRoundValidator.EXPECT().ValidateRoundInput(gomock.Any()).Return([]string{"Title is required", "Description is required", "Location is required", "Start time is required", "User ID is required"})
			},
			payload: roundevents.CreateRoundRequestedPayload{
				Title:       "",
				Description: "",
				Location:    "",
				StartTime:   "",
				UserID:      "",
				ChannelID:   "",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       "",
					ErrorMessage: []string{"Title is required", "Description is required", "Location is required", "Start time is required", "User ID is required"},
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid timezone",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundutil.MockTimeParserInterface) {
				mockTimeParser.EXPECT().ParseUserTimeInput(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("invalid timezone"))
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				mockRoundValidator.EXPECT().ValidateRoundInput(gomock.Any()).Return([]string{})
			},
			payload: roundevents.CreateRoundRequestedPayload{
				Title:       roundtypes.Title("Test Round"),
				Description: roundtypes.Description("Test Description"),
				StartTime:   "2024-01-01T12:00:00Z",
				Location:    roundtypes.Location("Test Location"),
				UserID:      "Test User",
				ChannelID:   "Test Channel",
				Timezone:    roundtypes.Timezone("Invalid/Timezone"),
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       "Test User",
					ErrorMessage: []string{"invalid timezone"},
				},
			},
			expectedError: nil,
		},
		{
			name: "start time in the past",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundutil.MockTimeParserInterface) {
				mockTimeParser.EXPECT().ParseUserTimeInput(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			mockRoundValidatorSetup: func(mockRoundValidator *roundutil.MockRoundValidator) {
				mockRoundValidator.EXPECT().ValidateRoundInput(gomock.Any()).Return([]string{})
			},
			payload: roundevents.CreateRoundRequestedPayload{
				Title:       roundtypes.Title("Test Round"),
				Description: roundtypes.Description("Test Description"),
				StartTime:   "2020-01-01T12:00:00Z",
				Location:    roundtypes.Location("Test Location"),
				UserID:      "Test User",
				ChannelID:   "Test Channel",
				Timezone:    roundtypes.Timezone("America/New_York"),
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundValidationFailedPayload{
					UserID:       "Test User",
					ErrorMessage: []string{"start time is in the past"},
				},
			},
			expectedError: nil,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)
			tt.mockTimeParserSetup(mockTimeParser)
			tt.mockRoundValidatorSetup(mockRoundValidator)

			// Initialize service with No-Op implementations
			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        metrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			result, err := s.ValidateAndProcessRound(ctx, tt.payload, mockTimeParser) // ← Capture result

			// Validate error presence
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
				return // Skip result validation if we expected an error
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
			}

			// Validate Success Results
			if tt.expectedResult.Success != nil {
				if result.Success == nil {
					t.Errorf("expected success result, got nil")
					return
				}

				// Cast the interface{} to the expected type
				expectedSuccess, ok := tt.expectedResult.Success.(*roundevents.RoundEntityCreatedPayload)
				if !ok {
					t.Errorf("expected success result is not RoundEntityCreatedPayload")
					return
				}

				actualSuccess, ok := result.Success.(*roundevents.RoundEntityCreatedPayload)
				if !ok {
					t.Errorf("actual success result is not RoundEntityCreatedPayload")
					return
				}

				// Now validate the fields
				if actualSuccess.Round.Title != expectedSuccess.Round.Title {
					t.Errorf("expected title %q, got %q", expectedSuccess.Round.Title, actualSuccess.Round.Title)
				}

				if actualSuccess.Round.CreatedBy != expectedSuccess.Round.CreatedBy {
					t.Errorf("expected created_by %q, got %q", expectedSuccess.Round.CreatedBy, actualSuccess.Round.CreatedBy)
				}

				if actualSuccess.Round.State != expectedSuccess.Round.State {
					t.Errorf("expected state %q, got %q", expectedSuccess.Round.State, actualSuccess.Round.State)
				}

				if actualSuccess.DiscordChannelID != expectedSuccess.DiscordChannelID {
					t.Errorf("expected channel_id %q, got %q", expectedSuccess.DiscordChannelID, actualSuccess.DiscordChannelID)
				}

				// Validate that ID was generated (if you add UUID generation)
				if actualSuccess.Round.ID == sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("expected Round.ID to be generated, got nil UUID")
				}
			}

			// Validate Failure Results
			if tt.expectedResult.Failure != nil {
				if result.Failure == nil {
					t.Errorf("expected failure result, got nil")
					return
				}

				// Cast the interface{} to the expected type
				expectedFailure, ok := tt.expectedResult.Failure.(*roundevents.RoundValidationFailedPayload)
				if !ok {
					t.Errorf("expected failure result is not RoundValidationFailedPayload")
					return
				}

				actualFailure, ok := result.Failure.(*roundevents.RoundValidationFailedPayload)
				if !ok {
					t.Errorf("actual failure result is not RoundValidationFailedPayload")
					return
				}

				if actualFailure.UserID != expectedFailure.UserID {
					t.Errorf("expected failure UserID %q, got %q", expectedFailure.UserID, actualFailure.UserID)
				}

				// Validate error messages
				if len(actualFailure.ErrorMessage) != len(expectedFailure.ErrorMessage) {
					t.Errorf("expected %d error messages, got %d", len(expectedFailure.ErrorMessage), len(actualFailure.ErrorMessage))
				} else {
					for i, expectedMsg := range expectedFailure.ErrorMessage {
						if i < len(actualFailure.ErrorMessage) && actualFailure.ErrorMessage[i] != expectedMsg {
							t.Errorf("expected error message[%d] %q, got %q", i, expectedMsg, actualFailure.ErrorMessage[i])
						}
					}
				}
			}

			// Ensure we don't have both success and failure
			if result.Success != nil && result.Failure != nil {
				t.Errorf("result should not have both success and failure")
			}

			// Ensure we have either success or failure (not neither)
			if result.Success == nil && result.Failure == nil {
				t.Errorf("result should have either success or failure, got neither")
			}
		})
	}
}

func TestRoundService_StoreRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := rounddb.NewMockRoundDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	metrics := &roundmetrics.NoOpMetrics{}

	tests := []struct {
		name           string
		mockDBSetup    func(*rounddb.MockRoundDB)
		payload        roundevents.RoundEntityCreatedPayload
		expectedResult RoundOperationResult
		expectedError  error
	}{
		{
			name: "store round successfully",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CreateRound(gomock.Any(), guildID, gomock.Any()).Return(nil)
			},
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					Title:        roundtypes.Title("Test Round"),
					Description:  roundtypes.DescriptionPtr("Test Description"),
					Location:     roundtypes.LocationPtr("Test Location"),
					StartTime:    (*sharedtypes.StartTime)(timePtr(time.Unix(1672531200, 0))),
					CreatedBy:    sharedtypes.DiscordID("12345678"),
					State:        roundtypes.RoundStateUpcoming,
					Participants: []roundtypes.Participant{},
					GuildID:      sharedtypes.GuildID("test-guild"),
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "test-guild",
			},
			expectedResult: RoundOperationResult{
				Success: &roundevents.RoundCreatedPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID:     sharedtypes.RoundID(uuid.New()),
						Title:       roundtypes.Title("Test Round"),
						Description: roundtypes.DescriptionPtr("Test Description"),
						Location:    roundtypes.LocationPtr("Test Location"),
						StartTime:   (*sharedtypes.StartTime)(timePtr(time.Unix(1672531200, 0))),
						UserID:      sharedtypes.DiscordID("12345678"),
						// GuildID intentionally omitted: not a field of BaseRoundPayload
					},
					ChannelID: "Test Channel",
				},
			},
			expectedError: nil,
		},
		{
			name: "store round fails",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CreateRound(gomock.Any(), guildID, gomock.Any()).Return(errors.New("database error"))
			},
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					Title:        roundtypes.Title("Test Round"),
					Description:  roundtypes.DescriptionPtr("Test Description"),
					Location:     roundtypes.LocationPtr("Test Location"),
					StartTime:    (*sharedtypes.StartTime)(timePtr(time.Unix(1672531200, 0))),
					CreatedBy:    sharedtypes.DiscordID("12345678"),
					State:        roundtypes.RoundStateUpcoming,
					Participants: []roundtypes.Participant{},
					GuildID:      sharedtypes.GuildID("test-guild"),
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "test-guild",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       "12345678",
					ErrorMessage: "failed to store round",
				},
			},
			expectedError: errors.New("failed to store round: database error"),
		},
		{
			name: "database error",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				guildID := sharedtypes.GuildID("test-guild")
				mockDB.EXPECT().CreateRound(gomock.Any(), guildID, gomock.Any()).Return(errors.New("database error"))
			},
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					Title:        roundtypes.Title("Test Round"),
					Description:  roundtypes.DescriptionPtr("Test Description"),
					Location:     roundtypes.LocationPtr("Test Location"),
					StartTime:    (*sharedtypes.StartTime)(timePtr(time.Unix(1672531200, 0))),
					CreatedBy:    sharedtypes.DiscordID("Test User"),
					State:        roundtypes.RoundStateUpcoming,
					Participants: []roundtypes.Participant{},
					GuildID:      sharedtypes.GuildID("test-guild"),
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "test-guild",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       "Test User",
					ErrorMessage: "failed to store round",
				},
			},
			expectedError: errors.New("failed to store round: database error"),
		},
		{
			name: "invalid round data",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			payload: roundevents.RoundEntityCreatedPayload{
				Round: roundtypes.Round{
					Title:        roundtypes.Title(""),
					Description:  roundtypes.DescriptionPtr(""),
					Location:     roundtypes.LocationPtr(""),
					StartTime:    (*sharedtypes.StartTime)(timePtr(time.Unix(0, 0))),
					CreatedBy:    sharedtypes.DiscordID(""),
					State:        roundtypes.RoundStateUpcoming,
					Participants: []roundtypes.Participant{},
					// GuildID intentionally omitted for invalid data
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "",
			},
			expectedResult: RoundOperationResult{
				Failure: &roundevents.RoundCreationFailedPayload{
					UserID:       "",
					ErrorMessage: "invalid round data",
				},
			},
			expectedError: errors.New("invalid round data"),
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockDBSetup(mockDB)

			// Initialize service with No-Op implementations
			s := &RoundService{
				RoundDB: mockDB,
				logger:  logger,
				metrics: metrics,
				tracer:  tracer,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			// Use a dummy guildID for testing
			guildID := sharedtypes.GuildID("test-guild")
			_, err := s.StoreRound(ctx, guildID, tt.payload)

			// Validate error presence
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error: %v, got: nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				if tt.expectedError != nil {
					if err == nil {
						t.Errorf("expected error: %v, got: nil", tt.expectedError)
					} else if err.Error() != tt.expectedError.Error() {
						t.Errorf("expected error message: %q, got: %q", tt.expectedError.Error(), err.Error())
					}
				} else if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}

			}
		})
	}
}
