// src/modules/leaderboard/leaderboard.module.ts
import { Module } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { LeaderboardResolver } from "./leaderboard.resolver";
import { DatabaseModule } from "../../db/database.module";

@Module({
  imports: [DatabaseModule],
  providers: [LeaderboardResolver, LeaderboardService],
  exports: [LeaderboardService],
})
export class LeaderboardModule {}
