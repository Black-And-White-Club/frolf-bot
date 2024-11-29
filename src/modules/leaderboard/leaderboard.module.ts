// src/modules/leaderboard/leaderboard.module.ts
import { Module } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { LeaderboardResolver } from "./leaderboard.resolver";
import { DatabaseModule } from "src/db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [
    DatabaseModule.forFeature(schema, "LEADERBOARD_DATABASE_CONNECTION"),
  ],
  providers: [LeaderboardResolver, LeaderboardService],
  exports: [LeaderboardService],
})
export class LeaderboardModule {}
