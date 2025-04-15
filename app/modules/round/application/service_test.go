package roundservice

import (
	"context"
	"errors"
	"reflect"
	"testing"

	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"

	"go.uber.org/mock/gomock"
)

func Test_serviceWrapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := &lokifrolfbot.NoOpLogger{}
	mockMetrics := &roundmetrics.NoOpMetrics{}
	mockTracer := tempofrolfbot.NewNoOpTracer()

	tests := []struct {
		name        string
		ctx         context.Context
		operation   string
		serviceFunc func() (RoundOperationResult, error)
		wantResult  RoundOperationResult
		wantErr     error
	}{
		{
			name:      "successful operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func() (RoundOperationResult, error) {
				return RoundOperationResult{
					Success: "test_success",
				}, nil
			},
			wantResult: RoundOperationResult{
				Success: "test_success",
			},
			wantErr: nil,
		},
		{
			name:      "failed operation",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func() (RoundOperationResult, error) {
				return RoundOperationResult{}, errors.New("test_error")
			},
			wantResult: RoundOperationResult{},
			wantErr:    errors.New("test_operation operation failed: test_error"),
		},
		{
			name:      "panic recovery",
			ctx:       context.Background(),
			operation: "test_operation",
			serviceFunc: func() (RoundOperationResult, error) {
				panic("test_panic")
			},
			wantResult: RoundOperationResult{},
			wantErr:    errors.New("Panic in test_operation: test_panic"),
		},
		{
			name:        "nil service function",
			ctx:         context.Background(),
			operation:   "test_operation",
			serviceFunc: nil,
			wantResult:  RoundOperationResult{},
			wantErr:     errors.New("service function is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := serviceWrapper(tt.ctx, tt.operation, tt.serviceFunc, mockLogger, mockMetrics, mockTracer)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("serviceWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr != nil {
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("serviceWrapper() error message = %q, want %q", err.Error(), tt.wantErr.Error())
				}
			}
			if !reflect.DeepEqual(gotResult.Success, tt.wantResult.Success) {
				t.Errorf("serviceWrapper() Success = %v, want %v", gotResult.Success, tt.wantResult.Success)
			}
			if !reflect.DeepEqual(gotResult.Failure, tt.wantResult.Failure) {
				t.Errorf("serviceWrapper() Failure = %v, want %v", gotResult.Failure, tt.wantResult.Failure)
			}
		})
	}
}
