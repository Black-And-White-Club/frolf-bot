// src/modules/leaderboard/leaderboard.resolver.ts
import { Injectable } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { UpdateTagDto } from "../../dto/leaderboard/update-tag.dto";
import { ReceiveScoresDto } from "../../dto/leaderboard/receive-scores.dto";
import { LinkTagDto } from "../../dto/leaderboard/link-tag.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { TagNumber } from "../../types.generated";

@Injectable()
export class LeaderboardResolver {
  constructor(private readonly leaderboardService: LeaderboardService) {}

  async getLeaderboard(page: any = 1, limit: any = 50) {
    const numericPage = parseInt(page, 10) || 1; // Default to 1 if parseInt fails or is NaN
    const numericLimit = parseInt(limit, 10) || 50; // Default to 50 if parseInt fails or is NaN

    console.log("Converted page:", numericPage, "Type:", typeof numericPage);
    console.log("Converted limit:", numericLimit, "Type:", typeof numericLimit);

    // Validate numbers
    if (numericPage <= 0 || numericLimit <= 0) {
      throw new Error("Page and limit must be positive numbers");
    }

    // Call the service with valid values
    return this.leaderboardService.getLeaderboard(numericPage, numericLimit);
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

  async updateTag(discordID: string, tagNumber: number): Promise<TagNumber> {
    try {
      // 1. Validate the input using the UpdateTagDto
      const updateTagDto = plainToClass(UpdateTagDto, { discordID, tagNumber });
      const errors = await validate(updateTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      // 2. Call the service to update the tag
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
      return await this.leaderboardService.processScores(
        receiveScoresDto.scores
      );
    } catch (error) {
      console.error("Error receiving scores:", error);
      throw new Error("Failed to receive scores");
    }
  }

  async manualTagUpdate(
    discordID: string,
    newTagNumber: number
  ): Promise<TagNumber> {
    try {
      // 1. Validate the input (you might have a ManualTagUpdateDto)

      // 2. Call updateTag to handle the update
      return await this.updateTag(discordID, newTagNumber);
    } catch (error) {
      console.error("Error during manual tag update:", error);
      throw new Error("Failed to update tag manually");
    }
  }

  async linkTag(discordID: string, newTagNumber: number): Promise<TagNumber> {
    try {
      // 1. Validate the input using LinkTagDto
      const linkTagDto = plainToClass(LinkTagDto, { discordID, newTagNumber });
      const errors = await validate(LinkTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      // 2. Call updateTag to handle the update
      return await this.updateTag(discordID, newTagNumber);
    } catch (error) {
      console.error("Error linking tag:", error);
      throw new Error("Failed to link tag");
    }
  }
}
