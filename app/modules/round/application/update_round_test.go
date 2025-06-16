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
	queuemocks "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue/mocks"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateAndProcessRoundUpdate(t *testing.T) { // ← Updated method name
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockTimeParser := roundtime.NewMockTimeParserInterface(ctrl) // ← Add mock time parser

	s := &RoundService{
		RoundDB:        mockDB,
		logger:         logger,
		metrics:        mockMetrics,
		tracer:         tracer,
		roundValidator: mockRoundValidator,
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	testRoundID := sharedtypes.RoundID(uuid.New())
	tests := []struct {
		name    string
		payload roundevents.UpdateRoundRequestedPayload // ← Changed to Discord payload type
		want    RoundOperationResult
		wantErr bool
	}{
		{
			name: "valid request",
			payload: roundevents.UpdateRoundRequestedPayload{ // ← Updated payload structure
				RoundID:  testRoundID,
				UserID:   sharedtypes.DiscordID("user123"),
				Title:    titlePtr("New Title"), // ← Pointer types
				Timezone: timezonePtr("America/Chicago"),
			},
			want: RoundOperationResult{
				Success: &roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with time parsing",
			payload: roundevents.UpdateRoundRequestedPayload{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     titlePtr("New Title"),
				StartTime: stringPtr("tomorrow at 2pm"),
				Timezone:  timezonePtr("America/Chicago"),
			},
			want: RoundOperationResult{
				Success: &roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
						// Don't set StartTime here - it will be set dynamically in the test
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid request - zero round ID",
			payload: roundevents.UpdateRoundRequestedPayload{
				RoundID: sharedtypes.RoundID(uuid.Nil),
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "validation failed: round ID cannot be zero; at least one field to update must be provided",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid request - no fields to update",
			payload: roundevents.UpdateRoundRequestedPayload{
				RoundID: testRoundID,
				UserID:  sharedtypes.DiscordID("user123"),
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "validation failed: at least one field to update must be provided",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid request - time parsing failed",
			payload: roundevents.UpdateRoundRequestedPayload{
				RoundID:   testRoundID,
				UserID:    sharedtypes.DiscordID("user123"),
				Title:     titlePtr("New Title"),
				StartTime: stringPtr("invalid time"),
				Timezone:  timezonePtr("America/Chicago"),
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "validation failed: time parsing failed: invalid time format",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up time parser expectations with a fixed time for testing
			var expectedParsedTime int64
			if tt.payload.StartTime != nil {
				if *tt.payload.StartTime == "tomorrow at 2pm" {
					// Use a FUTURE time for consistent testing
					expectedParsedTime = time.Now().Add(24 * time.Hour).Unix() // Tomorrow
					mockTimeParser.EXPECT().ParseUserTimeInput(*tt.payload.StartTime, *tt.payload.Timezone, gomock.Any()).Return(expectedParsedTime, nil)

					// Update the expected result with the actual parsed time
					if successPayload, ok := tt.want.Success.(*roundevents.RoundUpdateValidatedPayload); ok {
						expectedStartTime := sharedtypes.StartTime(time.Unix(expectedParsedTime, 0).UTC())
						successPayload.RoundUpdateRequestPayload.StartTime = &expectedStartTime
					}
				} else if *tt.payload.StartTime == "invalid time" {
					mockTimeParser.EXPECT().ParseUserTimeInput(*tt.payload.StartTime, *tt.payload.Timezone, gomock.Any()).Return(int64(0), errors.New("invalid time format"))
				}
			}

			// Updated: Now calls ValidateAndProcessRoundUpdate with timeParser
			got, err := s.ValidateAndProcessRoundUpdate(context.Background(), tt.payload, mockTimeParser)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ValidateAndProcessRoundUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, cmp.AllowUnexported(sharedtypes.StartTime{})); diff != "" {
				t.Errorf("RoundService.ValidateAndProcessRoundUpdate() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function for title pointers
func titlePtr(t string) *roundtypes.Title {
	title := roundtypes.Title(t)
	return &title
}

// Helper function for timezone pointers
func timezonePtr(t string) *roundtypes.Timezone {
	timezone := roundtypes.Timezone(t)
	return &timezone
}

func TestRoundService_UpdateRoundEntity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddb.NewMockRoundDB(ctrl)
	logger := loggerfrolfbot.NoOpLogger
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)

	s := &RoundService{
		RoundDB:        mockDB,
		logger:         logger,
		metrics:        mockMetrics,
		tracer:         tracer,
		roundValidator: mockRoundValidator,
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		payload roundevents.RoundUpdateValidatedPayload
		want    RoundOperationResult
		wantErr bool
	}{
		{
			name: "valid update",
			payload: roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
					RoundID: testRoundID,
					Title:   roundtypes.Title("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			want: RoundOperationResult{
				Success: &roundevents.RoundEntityUpdatedPayload{
					Round: roundtypes.Round{
						ID:    testRoundID,
						Title: roundtypes.Title("New Title"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid update - round not found",
			payload: roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
					RoundID: testRoundID,
					Title:   roundtypes.Title("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
					Error: "failed to update round in database: round not found",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid update - update failed",
			payload: roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
					RoundID: testRoundID,
					Title:   roundtypes.Title("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
					Error: "failed to update round in database: update failed",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "valid update":
				// Updated: Implementation only calls UpdateRound, not GetRound first
				mockDB.EXPECT().UpdateRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.RoundID, gomock.Any()).Return(&roundtypes.Round{
					ID:    tt.payload.RoundUpdateRequestPayload.RoundID,
					Title: tt.payload.RoundUpdateRequestPayload.Title,
				}, nil)
			case "invalid update - round not found":
				// Updated: Implementation calls UpdateRound which can return "round not found" error
				mockDB.EXPECT().UpdateRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.RoundID, gomock.Any()).Return(nil, errors.New("round not found"))
			case "invalid update - update failed":
				// Updated: Implementation calls UpdateRound which can return "update failed" error
				mockDB.EXPECT().UpdateRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.RoundID, gomock.Any()).Return(nil, errors.New("update failed"))
			}

			got, err := s.UpdateRoundEntity(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateRoundEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
				t.Errorf("RoundService.UpdateRoundEntity() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	// Use UTC time to match implementation
	testStartUpdateTime := sharedtypes.StartTime(time.Now().UTC().Add(2 * time.Hour))
	tests := []struct {
		name      string
		payload   roundevents.RoundScheduleUpdatePayload
		mockSetup func(*rounddb.MockRoundDB, *queuemocks.MockQueueService)
		want      RoundOperationResult
		wantErr   bool
	}{
		{
			name: "valid update",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			mockSetup: func(mockDB *rounddb.MockRoundDB, mockQueue *queuemocks.MockQueueService) {
				// Expect cancellation of existing jobs
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), testRoundID).Return(nil)

				// Expect GetEventMessageID call
				mockDB.EXPECT().GetEventMessageID(gomock.Any(), testRoundID).Return("event123", nil)

				// Expect GetRound call to get current round data
				mockDB.EXPECT().GetRound(gomock.Any(), testRoundID).Return(&roundtypes.Round{
					ID:    testRoundID,
					Title: roundtypes.Title("Old Title"),
				}, nil)

				// Expect reminder scheduling (use gomock.Any() for time since implementation uses time.Now())
				mockQueue.EXPECT().ScheduleRoundReminder(gomock.Any(), testRoundID, gomock.Any(), gomock.Any()).Return(nil)

				// Expect round start scheduling (use gomock.Any() for time since implementation uses time.Now())
				mockQueue.EXPECT().ScheduleRoundStart(gomock.Any(), testRoundID, gomock.Any(), gomock.Any()).Return(nil)
			},
			want: RoundOperationResult{
				Success: &roundevents.RoundScheduleUpdatePayload{
					RoundID:   testRoundID,
					Title:     roundtypes.Title("New Title"),
					Location:  roundtypes.LocationPtr("New Location"),
					StartTime: &testStartUpdateTime,
				},
			},
			wantErr: false,
		},
		{
			name: "error cancelling jobs",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			mockSetup: func(mockDB *rounddb.MockRoundDB, mockQueue *queuemocks.MockQueueService) {
				// Expect cancellation to fail
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), testRoundID).Return(errors.New("cancel jobs failed"))
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "failed to cancel existing scheduled jobs: cancel jobs failed",
				},
			},
			wantErr: false,
		},
		{
			name: "error getting event message ID",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			mockSetup: func(mockDB *rounddb.MockRoundDB, mockQueue *queuemocks.MockQueueService) {
				// Expect cancellation to succeed
				mockQueue.EXPECT().CancelRoundJobs(gomock.Any(), testRoundID).Return(nil)

				// Expect GetEventMessageID to fail
				mockDB.EXPECT().GetEventMessageID(gomock.Any(), testRoundID).Return("", errors.New("event message ID not found"))
			},
			want: RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "failed to get EventMessageID: event message ID not found",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := rounddb.NewMockRoundDB(ctrl)
			logger := loggerfrolfbot.NoOpLogger
			tracerProvider := noop.NewTracerProvider()
			tracer := tracerProvider.Tracer("test")
			mockMetrics := &roundmetrics.NoOpMetrics{}
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockQueue := queuemocks.NewMockQueueService(ctrl)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         logger,
				metrics:        mockMetrics,
				tracer:         tracer,
				roundValidator: mockRoundValidator,
				QueueService:   mockQueue,
				serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc(ctx)
				},
			}

			// Setup mocks
			tt.mockSetup(mockDB, mockQueue)

			got, err := s.UpdateScheduledRoundEvents(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want, cmpopts.EquateComparable(sharedtypes.StartTime{})); diff != "" {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
