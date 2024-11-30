// src/modules/leaderboard/leaderboard.module.ts
import { Module } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { LeaderboardController } from "./leaderboard.controller"; // Import LeaderboardController
import { DatabaseModule } from "src/db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [
    DatabaseModule.forFeature(schema, "LEADERBOARD_DATABASE_CONNECTION"),
  ],
  providers: [LeaderboardController, LeaderboardService], // Provide LeaderboardController
  exports: [LeaderboardService],
})
export class LeaderboardModule {}
