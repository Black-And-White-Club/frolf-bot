package graph_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph"
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
		us      *graph.UserServices
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "UserFound",
			us: &graph.UserServices{
				UserService: &services.UserService{
					GetUserFunc: func(ctx context.Context, discordID string) (*model.User, error) {
						return &model.User{ID: "1", DiscordID: discordID}, nil
					},
				},
			},
			args: args{
				ctx:       context.Background(),
				discordID: "12345",
			},
			want:    &model.User{ID: "1", DiscordID: "12345"},
			wantErr: false,
		},
		{
			name: "User  Not Found",
			us: &graph.UserServices{
				UserService: &services.UserService{
					GetUserFunc: func(ctx context.Context, discordID string) (*model.User, error) {
						return nil, errors.New("user not found")
					},
				},
			},
			args: args{
				ctx:       context.Background(),
				discordID: "67890",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.us.GetUser(tt.args.ctx, tt.args.discordID)
			if (err != nil) != tt.wantErr {
				t.Errorf("UserServices.GetUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserServices.GetUser () = %v, want %v", got, tt.want)
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
		r       *graph.Resolver
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "CreateUserSuccess",
			r: &graph.Resolver{
				UserServices: graph.UserServices{
					UserService: &services.UserService{
						CreateUserFunc: func(ctx context.Context, input model.UserInput) (*model.User, error) {
							return &model.User{ID: "1", DiscordID: input.DiscordID}, nil
						},
					},
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UserInput{
					DiscordID: "12345",
				},
			},
			want:    &model.User{ID: "1", DiscordID: "12345"},
			wantErr: false,
		},
		{
			name: "CreateUserFailure",
			r: &graph.Resolver{
				UserServices: graph.UserServices{
					UserService: &services.UserService{
						CreateUserFunc: func(ctx context.Context, input model.UserInput) (*model.User, error) {
							return nil, errors.New("failed to create user")
						},
					},
				},
			},
			args: args{
				ctx: context.Background(),
				input: model.UserInput{
					DiscordID: "67890",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.CreateUser(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.CreateUser  () error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.CreateUser  () = %v, want %v", got, tt.want)
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
		want *graph.Resolver
	}{
		{
			name: "CreateNewResolver",
			args: args{
				userService:        &services.UserService{},
				scoreService:       &services.ScoreService{},
				roundService:       &services.RoundService{},
				leaderboardService: &services.LeaderboardService{},
				db:                 &firestore.Client{},
			},
			want: &graph.Resolver{
				UserServices: graph.UserServices{
					UserService: &services.UserService{},
				},
				ScoringServices: graph.ScoringServices{
					ScoreService: &services.ScoreService{},
					RoundService: &services.RoundService{},
				},
				LeaderboardService: &services.LeaderboardService{},
				DB:                 &firestore.Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.NewResolver(tt.args.userService, tt.args.scoreService, tt.args.roundService, tt.args.leaderboardService, tt.args.db)
			if !reflect.DeepEqual(got, tt.want) {
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
		{
			name: "User IDFound",
			args: args{
				ctx: context.WithValue(context.Background(), graph.UserIDKey, "12345"), // Use the custom key from the graph package
			},
			want:    "12345",
			wantErr: false,
		},
		{
			name: "User IDNotFound",
			args: args{
				ctx: context.Background(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := graph.GetUserIDFromContext(tt.args.ctx) // Call the function from the graph package
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser IDFromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetUser IDFromContext() = %v, want %v", got, tt.want)
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
		r       *graph.Resolver
		args    args
		want    *model.User
		wantErr bool
	}{
		{
			name: "GetUserSuccess",
			r: &graph.Resolver{
				UserServices: graph.UserServices{
					UserService: &services.UserService{
						GetUserFunc: func(ctx context.Context, discordID string) (*model.User, error) {
							return &model.User{ID: "1", DiscordID: discordID}, nil
						},
					},
				},
			},
			args: args{
				ctx:       context.Background(),
				discordID: "12345",
			},
			want:    &model.User{ID: "1", DiscordID: "12345"},
			wantErr: false,
		},
		{
			name: "GetUserNotFound",
			r: &graph.Resolver{
				UserServices: graph.UserServices{
					UserService: &services.UserService{
						GetUserFunc: func(ctx context.Context, discordID string) (*model.User, error) {
							return nil, errors.New("user not found")
						},
					},
				},
			},
			args: args{
				ctx:       context.Background(),
				discordID: "67890",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetUser(tt.args.ctx, tt.args.discordID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetUser () error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.GetUser () = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestResolver_GetLeaderboard(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		r       *graph.Resolver
		args    args
		want    *model.Leaderboard
		wantErr bool
	}{
		{
			name: "GetLeaderboardSuccess",
			r: &graph.Resolver{
				LeaderboardService: &services.LeaderboardService{
					GetLeaderboardFunc: func(ctx context.Context) (*model.Leaderboard, error) {
						return &model.Leaderboard{
							Users:      []*model.User{{ID: "1", DiscordID: "12345"}},  // Use pointers here
							Placements: []*model.Tag{{ID: "1", TagNumber: intPtr(1)}}, // Use pointers here
						}, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    &model.Leaderboard{Users: []*model.User{{ID: "1", DiscordID: "12345"}}, Placements: []*model.Tag{{ID: "1", TagNumber: intPtr(1)}}}, // Use pointers here
			wantErr: false,
		},
		{
			name: "Get Leaderboard Failure",
			r: &graph.Resolver{
				LeaderboardService: &services.LeaderboardService{
					GetLeaderboardFunc: func(ctx context.Context) (*model.Leaderboard, error) {
						return nil, errors.New("failed to get leaderboard")
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
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
		r       *graph.Resolver
		args    args
		want    []*model.Round
		wantErr bool
	}{
		{
			name: "Get Rounds Success",
			r: &graph.Resolver{
				ScoringServices: graph.ScoringServices{
					RoundService: &services.RoundService{
						GetRoundsFunc: func(ctx context.Context, limit *int, offset *int) ([]*model.Round, error) {
							return []*model.Round{{ID: "1", Title: "Round 1", Location: "Location 1", Finalized: false, Participants: []*model.Participant{}, Scores: []*model.Score{}, EditHistory: []*model.EditLog{}, CreatorID: "creatorID"}}, nil
						},
					},
				},
			},
			args: args{
				ctx:    context.Background(),
				limit:  nil,
				offset: nil,
			},
			want:    []*model.Round{{ID: "1", Title: "Round 1", Location: "Location 1", Finalized: false, Participants: []*model.Participant{}, Scores: []*model.Score{}, EditHistory: []*model.EditLog{}, CreatorID: "creatorID"}},
			wantErr: false,
		},
		{
			name: "Get Rounds Failure",
			r: &graph.Resolver{
				ScoringServices: graph.ScoringServices{
					RoundService: &services.RoundService{
						GetRoundsFunc: func(ctx context.Context, limit *int, offset *int) ([]*model.Round, error) {
							return nil, errors.New("failed to get rounds")
						},
					},
				},
			},
			args: args{
				ctx:    context.Background(),
				limit:  nil,
				offset: nil,
			},
			want:    nil,
			wantErr: true,
		},
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
		r       *graph.Resolver
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "Get User Score Success",
			r: &graph.Resolver{
				ScoringServices: graph.ScoringServices{
					ScoreService: &services.ScoreService{
						GetUserScoreFunc: func(ctx context.Context, userID string) (int, error) {
							return 100, nil
						},
					},
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: "1",
			},
			want:    100,
			wantErr: false,
		},
		{
			name: "Get User Score Failure",
			r: &graph.Resolver{
				ScoringServices: graph.ScoringServices{
					ScoreService: &services.ScoreService{
						GetUserScoreFunc: func(ctx context.Context, userID string) (int, error) {
							return 0, errors.New("failed to get user score")
						},
					},
				},
			},
			args: args{
				ctx:    context.Background(),
				userID: "2",
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.GetUserScore(tt.args.ctx, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolver.GetUser Score() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Resolver.GetUser Score() = %v, want %v", got, tt.want)
			}
		})
	}
}
