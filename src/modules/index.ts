// src/modules/index.ts
export * from "./user";
export * from "./score";
export * from "./round";
export * from "./leaderboard";

import { UserResolver } from "./user/user.resolver";
import { ScoreResolver } from "./score/score.resolver";
import { RoundResolver } from "./round/round.resolver";
import { LeaderboardResolver } from "./leaderboard/leaderboard.resolver";
import { UserService } from "./user/user.service";
import { ScoreService } from "./score/score.service";
import { RoundService } from "./round/round.service";
import { LeaderboardService } from "./leaderboard/leaderboard.service";
import { db } from "../database";

export function createResolvers(db: any) {
  const userResolver = new UserResolver(new UserService(db));
  const scoreResolver = new ScoreResolver(new ScoreService(db));
  const roundResolver = new RoundResolver(
    new RoundService(db),
    new LeaderboardService(db)
  );
  const leaderboardResolver = new LeaderboardResolver(
    new LeaderboardService(db)
  );

  return {
    Query: {
      getUser: userResolver.getUser.bind(userResolver),
      getUserScore: scoreResolver.getUserScore.bind(scoreResolver),
      getScoresForRound: scoreResolver.getScoresForRound.bind(scoreResolver),
      getRounds: roundResolver.getRounds.bind(roundResolver),
      getRound: roundResolver.getRound.bind(roundResolver),
      getLeaderboard: leaderboardResolver.getLeaderboard.bind(
        leaderboardResolver
      ),
      getUserTag: leaderboardResolver.getUserTag.bind(leaderboardResolver),
    },
    Mutation: {
      createUser: userResolver.createUser.bind(userResolver),
      updateUser: userResolver.updateUser.bind(userResolver),
      updateScore: scoreResolver.updateScore.bind(scoreResolver),
      processScores: scoreResolver.processScores.bind(scoreResolver),
      scheduleRound: roundResolver.scheduleRound.bind(roundResolver),
      joinRound: roundResolver.joinRound.bind(roundResolver),
      editRound: roundResolver.editRound.bind(roundResolver),
      submitScore: roundResolver.submitScore.bind(roundResolver),
      updateTag: leaderboardResolver.updateTag.bind(leaderboardResolver),
      receiveScores: leaderboardResolver.receiveScores.bind(
        leaderboardResolver
      ),
      manualTagUpdate: leaderboardResolver.manualTagUpdate.bind(
        leaderboardResolver
      ),
      linkTag: leaderboardResolver.linkTag.bind(leaderboardResolver),
      // ... add other Mutation resolvers as needed
    },
  } as const;
}
