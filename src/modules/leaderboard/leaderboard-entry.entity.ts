// src/modules/leaderboard/entities/leaderboard-entry.entity.ts
import { IsString, IsInt } from "class-validator";

export class LeaderboardEntry {
  @IsString()
  discordID!: string;

  @IsInt()
  tagNumber!: number;
}
