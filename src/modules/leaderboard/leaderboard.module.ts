import { Module } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { LeaderboardResolver } from "./leaderboard.resolver";
@Module({
  providers: [LeaderboardService, LeaderboardResolver],
  exports: [LeaderboardService],
})
export class LeaderboardModule {}
