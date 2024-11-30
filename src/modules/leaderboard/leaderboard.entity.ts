// src/modules/leaderboard/entities/leaderboard.entity.ts
import { IsArray, ValidateNested } from "class-validator";
import { Type } from "class-transformer";
import { LeaderboardEntry } from "./leaderboard-entry.entity"; // Assuming you'll create this entity

export class Leaderboard {
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => LeaderboardEntry)
  leaderboardData!: LeaderboardEntry[];
}
