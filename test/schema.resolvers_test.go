package graph

import (
	"context"
	"reflect"
	"testing"

	"github.com/romero-jace/tcr-bot/graph/model"
)

func Test_mutationResolver_CreateUser(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.UserInput
	}
	tests := []struct {
		name    string
		r       *mutationResolver
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
				t.Errorf("mutationResolver.CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.CreateUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_ScheduleRound(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.RoundInput
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.ScheduleRound(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.ScheduleRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.ScheduleRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_JoinRound(t *testing.T) {
	type args struct {
		ctx   context.Context
		input model.JoinRoundInput
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.JoinRound(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.JoinRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.JoinRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_SubmitScore(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
		userID  string
		score   int
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.SubmitScore(tt.args.ctx, tt.args.roundID, tt.args.userID, tt.args.score)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.SubmitScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.SubmitScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_FinalizeRound(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.FinalizeRound(tt.args.ctx, tt.args.roundID)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.FinalizeRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.FinalizeRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_EditRound(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
		input   model.RoundInput
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    *model.Round
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.EditRound(tt.args.ctx, tt.args.roundID, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.EditRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mutationResolver.EditRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mutationResolver_DeleteRound(t *testing.T) {
	type args struct {
		ctx     context.Context
		roundID string
	}
	tests := []struct {
		name    string
		r       *mutationResolver
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.DeleteRound(tt.args.ctx, tt.args.roundID)
			if (err != nil) != tt.wantErr {
				t.Errorf("mutationResolver.DeleteRound() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("mutationResolver.DeleteRound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryResolver_GetUser(t *testing.T) {
	type args struct {
		ctx       context.Context
		discordID string
	}
	tests := []struct {
		name    string
		r       *queryResolver
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
				t.Errorf("queryResolver.GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queryResolver.GetUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryResolver_GetLeaderboard(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		r       *queryResolver
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
				t.Errorf("queryResolver.GetLeaderboard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queryResolver.GetLeaderboard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryResolver_GetRounds(t *testing.T) {
	type args struct {
		ctx    context.Context
		limit  *int
		offset *int
	}
	tests := []struct {
		name    string
		r       *queryResolver
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
				t.Errorf("queryResolver.GetRounds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queryResolver.GetRounds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryResolver_GetUserScore(t *testing.T) {
	type args struct {
		ctx    context.Context
		userID string
	}
	tests := []struct {
		name    string
		r       *queryResolver
		args    args
		want    *int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetUserScore(tt.args.ctx, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("queryResolver.GetUserScore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("queryResolver.GetUserScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Mutation(t *testing.T) {
	tests := []struct {
		name string
		r    *Resolver
		want MutationResolver
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Mutation(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.Mutation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Query(t *testing.T) {
	tests := []struct {
		name string
		r    *Resolver
		want QueryResolver
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Query(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.Query() = %v, want %v", got, tt.want)
			}
		})
	}
}
