// graph/services/round_service.go
package services

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
)

func TestNewRoundService(t *testing.T) {
	type args struct {
		client       *firestore.Client
		scoreService *ScoreService
	}
	tests := []struct {
		name string
		args args
		want *RoundService
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRoundService(tt.args.client, tt.args.scoreService); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRoundService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateID(); got != tt.want {
				t.Errorf("generateID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_normalizeDateTime(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := normalizeDateTime(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeDateTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("normalizeDateTime() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("normalizeDateTime() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestRoundService_ScheduleRound(t *testing.T) {
	type args struct {
		ctx       context.Context
		input     model.RoundInput
		creatorID string
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.ScheduleRound(tt.args.ctx, tt.args.input, tt.args.creatorID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.ScheduleRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.ScheduleRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_SubmitScore(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
		scores  map[string]string
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.SubmitScore(tt.args.ctx, tt.args.roundID, tt.args.scores)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.SubmitScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.SubmitScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_JoinRound(t *testing.T) {
	type args struct {
		ctx      context.Context
		roundID  string
		userID   string
		response model.Response
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.JoinRound(tt.args.ctx, tt.args.roundID, tt.args.userID, tt.args.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.JoinRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.JoinRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_GetRounds(t *testing.T) {
	type args struct {
		ctx    context.Context
		limit  *int
		offset *int
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    []*model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetRounds(tt.args.ctx, tt.args.limit, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.GetRounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.GetRounds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_GetRoundByID(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetRoundByID(tt.args.ctx, tt.args.roundID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.GetRoundByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.GetRoundByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_EditRound(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
		userID  string
		input   model.RoundInput
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.EditRound(tt.args.ctx, tt.args.roundID, tt.args.userID, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.EditRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.EditRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundService_DeleteRound(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
		userID  string
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.DeleteRound(tt.args.ctx, tt.args.roundID, tt.args.userID); (err != nil) != tt.wantErr {
				t.Errorf("RoundService.DeleteRound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundService_FinalizeRound(t *testing.T) {
	type args struct {
		ctx      context.Context
		roundID  string
		editorID string
	}
	tests := []struct {
		name    string
		s       *RoundService
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.FinalizeRound(tt.args.ctx, tt.args.roundID, tt.args.editorID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RoundService.FinalizeRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoundService.FinalizeRound() = %v, want %v", got, tt.want)
			}
		})
	}
}
