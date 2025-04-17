package leaderboardservice

import (
	"context"
	"fmt"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddbtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func BenchmarkUpdateLeaderboardSmallInput(b *testing.B) {
	sortedParticipantTags := []string{"1:user1", "2:user2", "3:user3"}

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(&leaderboarddbtypes.Leaderboard{
		LeaderboardData: []leaderboardtypes.LeaderboardEntry{
			{TagNumber: 1, UserID: "user1"},
			{TagNumber: 2, UserID: "user2"},
			{TagNumber: 3, UserID: "user3"},
		},
	}, nil).AnyTimes()
	mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Setup logger and tracer outside of struct literal
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	service := &LeaderboardService{
		LeaderboardDB: mockDB,
		logger:        loggerfrolfbot.NoOpLogger,
		tracer:        tracer,
		metrics:       &leaderboardmetrics.NoOpMetrics{},
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.UpdateLeaderboard(context.Background(), sharedtypes.RoundID(uuid.New()), sortedParticipantTags)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateLeaderboardMediumInput(b *testing.B) {
	// Create a medium-sized test leaderboard and sorted participant tags
	leaderboard := &leaderboarddbtypes.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 100),
	}
	for i := range leaderboard.LeaderboardData {
		leaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(i + 1),
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
		}
	}
	sortedParticipantTags := make([]string, 100)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("%d:user%d", i+1, i+1)
	}

	// Setup logger and tracer outside of struct literal
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")
	// Create a test service instance with mock dependencies
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(leaderboard, nil).AnyTimes()
	mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	service := &LeaderboardService{
		LeaderboardDB: mockDB,
		logger:        loggerfrolfbot.NoOpLogger,
		tracer:        tracer,
		metrics:       &leaderboardmetrics.NoOpMetrics{},
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.UpdateLeaderboard(context.Background(), sharedtypes.RoundID(uuid.New()), sortedParticipantTags)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateLeaderboardLargeInput(b *testing.B) {
	// Create a large test leaderboard and sorted participant tags
	leaderboard := &leaderboarddbtypes.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 1000),
	}
	for i := range leaderboard.LeaderboardData {
		leaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(i + 1),
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("user%d", i+1)),
		}
	}
	sortedParticipantTags := make([]string, 1000)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("%d:user%d", i+1, i+1)
	}

	// Setup logger and tracer outside of struct literal
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	// Create a test service instance with mock dependencies
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockDB := leaderboarddb.NewMockLeaderboardDB(ctrl)
	mockDB.EXPECT().GetActiveLeaderboard(gomock.Any()).Return(leaderboard, nil).AnyTimes()
	mockDB.EXPECT().UpdateLeaderboard(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	service := &LeaderboardService{
		LeaderboardDB: mockDB,
		logger:        loggerfrolfbot.NoOpLogger,
		tracer:        tracer,
		metrics:       &leaderboardmetrics.NoOpMetrics{},
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.UpdateLeaderboard(context.Background(), sharedtypes.RoundID(uuid.New()), sortedParticipantTags)
		if err != nil {
			b.Fatal(err)
		}
	}
}
