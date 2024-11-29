// src/modules/leaderboard/leaderboard.resolver.ts
import { Injectable } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { UpdateTagDto } from "../../dto/leaderboard/update-tag.dto";
import { ReceiveScoresDto } from "../../dto/leaderboard/receive-scores.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Injectable()
export class LeaderboardResolver {
  constructor(private readonly leaderboardService: LeaderboardService) {}

  async getLeaderboard() {
    return this.leaderboardService.getLeaderboard();
  }

  async getUserTag(discordID: string) {
    try {
      const tag = await this.leaderboardService.getUserTag(discordID);
      if (!tag) {
        throw new Error("Tag not found for the provided discordID");
      }
      return tag;
    } catch (error) {
      console.error("Error fetching user tag:", error);
      throw new Error("Failed to fetch user tag");
    }
  }

  async updateTag(discordID: string, tagNumber: number) {
    try {
      const updateTagDto = plainToClass(UpdateTagDto, { discordID, tagNumber });
      const errors = await validate(updateTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      return await this.leaderboardService.updateTag(discordID, tagNumber);
    } catch (error) {
      console.error("Error updating tag:", error);
      throw new Error("Failed to update tag");
    }
  }

  async receiveScores(
    scores: Array<{ score: number; discordID: string; tagNumber?: number }>
  ) {
    try {
      const receiveScoresDto = plainToClass(ReceiveScoresDto, { scores });
      const errors = await validate(receiveScoresDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      await this.leaderboardService.processScores(receiveScoresDto.scores);

      // Return the updated leaderboard
      return this.leaderboardService.getLeaderboard();
    } catch (error) {
      console.error("Error receiving scores:", error);
      throw new Error("Failed to receive scores");
    }
  }
}
