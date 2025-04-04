package roundservice

import (
	"context"
	"errors"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
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
	mockTimeParser := roundtime.NewMockTimeParserInterface(ctrl)
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name                    string
		mockDBSetup             func(*rounddb.MockRoundDB)
		mockTimeParserSetup     func(*roundtime.MockTimeParserInterface)
		mockRoundValidatorSetup func(*roundutil.MockRoundValidator)
		payload                 roundevents.CreateRoundRequestedPayload
		expectedResult          RoundOperationResult
		expectedError           error
	}{
		{
			name: "valid round",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundtime.MockTimeParserInterface) {
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
			mockTimeParserSetup: func(mockTimeParser *roundtime.MockTimeParserInterface) {
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
			expectedError: errors.New("validation failed: [Title is required Description is required Location is required Start time is required User ID is required]"),
		},
		{
			name: "invalid timezone",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundtime.MockTimeParserInterface) {
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
			expectedError: errors.New("time parsing failed: invalid timezone"),
		},
		{
			name: "start time in the past",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
			},
			mockTimeParserSetup: func(mockTimeParser *roundtime.MockTimeParserInterface) {
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
			expectedError: errors.New("validation failed: [start time is in the past]"),
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
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			_, err := s.ValidateAndProcessRound(ctx, tt.payload, mockTimeParser)

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

func TestRoundService_StoreRound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	// Mock dependencies
	mockDB := rounddb.NewMockRoundDB(ctrl)

	// No-Op implementations for logging, metrics, and tracing
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &roundmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

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
				mockDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(nil)
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
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "",
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
					},
					ChannelID: "Test Channel",
				},
			},
			expectedError: nil,
		},
		{
			name: "store round fails",
			mockDBSetup: func(mockDB *rounddb.MockRoundDB) {
				mockDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
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
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "",
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
				mockDB.EXPECT().CreateRound(gomock.Any(), gomock.Any()).Return(errors.New("database error"))
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
				},
				DiscordChannelID: "Test Channel",
				DiscordGuildID:   "",
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
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			_, err := s.StoreRound(ctx, tt.payload)

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
