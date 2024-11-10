package services

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
)

func TestNewLeaderboardService(t *testing.T) {
	type args struct {
		client *firestore.Client
	}
	tests := []struct {
		name string
		args args
		want *LeaderboardService
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLeaderboardService(tt.args.client); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLeaderboardService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardService_GetLeaderboard(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		s       *LeaderboardService
		args    args
		want    *model.Leaderboard
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.GetLeaderboard(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.GetLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LeaderboardService.GetLeaderboard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardService_getAllUsers(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		s       *LeaderboardService
		args    args
		want    []*model.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.getAllUsers(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.getAllUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LeaderboardService.getAllUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardService_getPlacements(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		s       *LeaderboardService
		args    args
		want    []*model.Tag
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.s.getPlacements(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.getPlacements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LeaderboardService.getPlacements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLeaderboardService_UpdateLeaderboard(t *testing.T) {
	type args struct {
		ctx          context.Context
		userID       string
		newPlacement *model.Tag
	}
	tests := []struct {
		name    string
		s       *LeaderboardService
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.UpdateLeaderboard(tt.args.ctx, tt.args.userID, tt.args.newPlacement); (err != nil) != tt.wantErr {
				t.Errorf("LeaderboardService.UpdateLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLeaderboardService_hasPermission(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		s    *LeaderboardService
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.hasPermission(tt.args.userID); got != tt.want {
				t.Errorf("LeaderboardService.hasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}
