package graph

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
	"github.com/romero-jace/tcr-bot/graph/services"
)

func TestUserServices_GetUser(t *testing.T) {
	type args struct {
		ctx       context.Context
		discordID string
	}
	tests := []struct {
		name    string
		us      *UserServices
		args    args
		want    *model.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.us.GetUser(tt.args.ctx, tt.args.discordID)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserServices.GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserServices.GetUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_CreateUser(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UserInput
	}
	tests := []struct {
		name    string
		r       *Resolver
		args    args
		want    *model.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.CreateUser(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.CreateUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewResolver(t *testing.T) {
	type args struct {
		userService        *services.UserService
		scoreService       *services.ScoreService
		roundService       *services.RoundService
		leaderboardService *services.LeaderboardService
		db                 *firestore.Client
	}
	tests := []struct {
		name string
		args args
		want *Resolver
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewResolver(tt.args.userService, tt.args.scoreService, tt.args.roundService, tt.args.leaderboardService, tt.args.db); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewResolver() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUserIDFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getUserIDFromContext(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("getUserIDFromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getUserIDFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_GetUser(t *testing.T) {
	type args struct {
		ctx       context.Context
		discordID string
	}
	tests := []struct {
		name    string
		r       *Resolver
		args    args
		want    *model.User
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetUser(tt.args.ctx, tt.args.discordID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.GetUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_GetLeaderboard(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		r       *Resolver
		args    args
		want    *model.Leaderboard
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetLeaderboard(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.GetLeaderboard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_GetRounds(t *testing.T) {
	type args struct {
		ctx    context.Context
		limit  *int
		offset *int
	}
	tests := []struct {
		name    string
		r       *Resolver
		args    args
		want    []*model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetRounds(tt.args.ctx, tt.args.limit, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetRounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.GetRounds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_GetUserScore(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}
	tests := []struct {
		name    string
		r       *Resolver
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetUserScore(tt.args.ctx, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetUserScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolver.GetUserScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
