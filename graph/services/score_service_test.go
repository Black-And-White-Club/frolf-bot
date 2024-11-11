// graph/services/score_service.go
package services

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
)

func TestNewScoreService(t *testing.T) {
	type args struct {
		client *firestore.Client
	}
	tests := []struct {
		name string
		args args
		want *ScoreService
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewScoreService(tt.args.client); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewScoreService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreService_SubmitScore(t *testing.T) {
	type args struct {
		ctx    context.Context
		round  *model.Round
		scores map[string]string
	}
	tests := []struct {
		name    string
		s       *ScoreService
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.SubmitScore(tt.args.ctx, tt.args.round, tt.args.scores); (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.SubmitScore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScoreService_ProcessScoring(t *testing.T) {
	type args struct {
		round *model.Round
	}
	tests := []struct {
		name    string
		s       *ScoreService
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.ProcessScoring(tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.ProcessScoring() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isValidGolfScore(t *testing.T) {
	type args struct {
		score string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidGolfScore(tt.args.score); got != tt.want {
				t.Errorf("isValidGolfScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreService_GetUserScore(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}
	tests := []struct {
		name    string
		s       *ScoreService
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetUserScore(tt.args.ctx, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.GetUserScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ScoreService.GetUserScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreService_EditScore(t *testing.T) {
	type args struct {
		ctx         context.Context
		round       *model.Round
		userID      string
		newScoreStr string
	}
	tests := []struct {
		name    string
		s       *ScoreService
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.EditScore(tt.args.ctx, tt.args.round, tt.args.userID, tt.args.newScoreStr); (err != nil) != tt.wantErr {
				t.Errorf("ScoreService.EditScore() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
