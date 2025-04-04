package roundservice

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/mocks"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundService_ValidateRoundUpdateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	s := &RoundService{
		RoundDB:        mockDB,
		logger:         mockLogger,
		metrics:        mockMetrics,
		tracer:         mockTracer,
		roundValidator: mockRoundValidator,
		EventBus:       mockEventBus,
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
			return serviceFunc()
		},
	}

	testRoundID := sharedtypes.RoundID(uuid.New())
	tests := []struct {
		name    string
		payload roundevents.RoundUpdateRequestPayload
		want    RoundOperationResult
		wantErr bool
	}{
		{
			name: "valid request",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: testRoundID,
					Title:   roundtypes.Title("New Title"),
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			want: RoundOperationResult{
				Success: roundevents.RoundUpdateValidatedPayload{
					RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
							Title:   roundtypes.Title("New Title"),
							UserID:  sharedtypes.DiscordID("user123"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid request - zero round ID",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: sharedtypes.RoundID(uuid.Nil),
				},
			},
			want: RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: sharedtypes.RoundID(uuid.Nil),
						},
					},
					Error: "validation errors: round ID cannot be zero; at least one field to update must be provided",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid request - no fields to update",
			payload: roundevents.RoundUpdateRequestPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID: testRoundID,
					UserID:  sharedtypes.DiscordID("user123"),
				},
			},
			want: RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
							UserID:  sharedtypes.DiscordID("user123"),
						},
					},
					Error: "validation errors: at least one field to update must be provided",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.ValidateRoundUpdateRequest(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ValidateRoundUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.ValidateRoundUpdateRequest() = \n%+v, want \n%+v", got, tt.want)
			}
		})
	}
}

func TestRoundService_UpdateRoundEntity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := rounddb.NewMockRoundDB(ctrl)
	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()
	mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
	mockEventBus := eventbus.NewMockEventBus(ctrl)

	s := &RoundService{
		RoundDB:        mockDB,
		logger:         mockLogger,
		metrics:        mockMetrics,
		tracer:         mockTracer,
		roundValidator: mockRoundValidator,
		EventBus:       mockEventBus,
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
			return serviceFunc()
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
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
				},
			},
			want: RoundOperationResult{
				Success: roundevents.RoundEntityUpdatedPayload{
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
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
				},
			},
			want: RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
							Title:   roundtypes.Title("New Title"),
							UserID:  sharedtypes.DiscordID("user123"),
						},
					},
					Error: "round not found",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update - update failed",
			payload: roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: roundevents.RoundUpdateRequestPayload{
					BaseRoundPayload: roundtypes.BaseRoundPayload{
						RoundID: testRoundID,
						Title:   roundtypes.Title("New Title"),
						UserID:  sharedtypes.DiscordID("user123"),
					},
				},
			},
			want: RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &roundevents.RoundUpdateRequestPayload{
						BaseRoundPayload: roundtypes.BaseRoundPayload{
							RoundID: testRoundID,
							Title:   roundtypes.Title("New Title"),
							UserID:  sharedtypes.DiscordID("user123"),
						},
					},
					Error: "update failed",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.name {
			case "valid update":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID).Return(&roundtypes.Round{
					ID:    tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID,
					Title: tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.Title,
				}, nil)
				mockDB.EXPECT().UpdateRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID, gomock.Any()).Return(nil)
			case "invalid update - round not found":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID).Return(nil, errors.New("round not found"))
			case "invalid update - update failed":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID).Return(&roundtypes.Round{
					ID:    tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID,
					Title: tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.Title,
				}, nil)
				mockDB.EXPECT().UpdateRound(gomock.Any(), tt.payload.RoundUpdateRequestPayload.BaseRoundPayload.RoundID, gomock.Any()).Return(errors.New("update failed"))
			}

			got, err := s.UpdateRoundEntity(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateRoundEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.UpdateRoundEntity() = \n%+v, want \n%+v", got, tt.want)
			}
		})
	}
}

func TestRoundService_UpdateScheduledRoundEvents(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testStartUpdateTime := sharedtypes.StartTime(time.Now().Add(2 * time.Hour))
	tests := []struct {
		name    string
		payload roundevents.RoundScheduleUpdatePayload
		want    RoundOperationResult
		wantErr bool
	}{
		{
			name: "valid update",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			want: RoundOperationResult{
				Success: roundevents.RoundStoredPayload{
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
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			want: RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "round not found",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid update - cancel scheduled events failed",
			payload: roundevents.RoundScheduleUpdatePayload{
				RoundID:   testRoundID,
				Title:     roundtypes.Title("New Title"),
				StartTime: &testStartUpdateTime,
				Location:  roundtypes.LocationPtr("New Location"),
			},
			want: RoundOperationResult{
				Success: roundevents.RoundStoredPayload{
					Round: roundtypes.Round{
						ID:    testRoundID,
						Title: roundtypes.Title("New Title"),
					},
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
			mockLogger := &lokifrolfbot.NoOpLogger{}
			mockMetrics := &roundmetrics.NoOpMetrics{}
			mockTracer := tempofrolfbot.NewNoOpTracer()
			mockRoundValidator := roundutil.NewMockRoundValidator(ctrl)
			mockEventBus := eventbus.NewMockEventBus(ctrl)

			s := &RoundService{
				RoundDB:        mockDB,
				logger:         mockLogger,
				metrics:        mockMetrics,
				tracer:         mockTracer,
				roundValidator: mockRoundValidator,
				EventBus:       mockEventBus,
				serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error) {
					return serviceFunc()
				},
			}

			switch tt.name {
			case "valid update":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundID).Return(&roundtypes.Round{
					ID:    tt.payload.RoundID,
					Title: tt.payload.Title,
				}, nil)
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), tt.payload.RoundID).Return(nil)
			case "invalid update - round not found":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundID).Return(nil, errors.New("round not found"))
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), tt.payload.RoundID).Return(nil)
			case "invalid update - cancel scheduled events failed":
				mockDB.EXPECT().GetRound(gomock.Any(), tt.payload.RoundID).Return(&roundtypes.Round{
					ID:    tt.payload.RoundID,
					Title: tt.payload.Title,
				}, nil)
				mockEventBus.EXPECT().CancelScheduledMessage(gomock.Any(), tt.payload.RoundID).Return(errors.New("cancel scheduled events failed"))
			}

			got, err := s.UpdateScheduledRoundEvents(context.Background(), tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.UpdateScheduledRoundEvents() = \n%+v, want \n%+v", got, tt.want)
			}
		})
	}
}
