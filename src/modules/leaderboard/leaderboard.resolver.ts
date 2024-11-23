import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { LeaderboardService } from "../services/LeaderboardService";
import { UpdateTagDto } from "../dto/update-tag.dto"; // DTO for updating tags
import { ReceiveScoresDto } from "../dto/receive-scores.dto"; // DTO for receiving scores
import { LinkTagDto } from "../dto/link-tag.dto"; // DTO for linking tags

export const LeaderboardResolver = {
  Query: {
    async getLeaderboard(
      _: any,
      args: { page?: number; limit?: number },
      context: { leaderboardService: LeaderboardService }
    ) {
      const { page = 1, limit = 50 } = args; // Default to page 1 and limit of 50
      return await context.leaderboardService.getLeaderboard(page, limit);
    },

    async getUserTag(
      _: any,
      args: { discordID: string },
      context: { leaderboardService: LeaderboardService }
    ) {
      const tag = await context.leaderboardService.getUserTag(args.discordID);
      if (!tag) {
        throw new Error("Tag not found for the provided discordID");
      }
      return tag;
    },
  },
  Mutation: {
    async updateTag(
      _: any,
      args: { discordID: string; tagNumber: number },
      context: { leaderboardService: LeaderboardService }
    ) {
      const updateTagDto = plainToClass(UpdateTagDto, {
        discordID: args.discordID,
        tagNumber: args.tagNumber,
      });
      const errors = await validate(updateTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      return await context.leaderboardService.updateTag(
        updateTagDto.discordID,
        updateTagDto.tagNumber
      );
    },

    async receiveScores(
      _: any,
      args: {
        scores: Array<{ score: number; discordID: string; tagNumber?: number }>;
      },
      context: { leaderboardService: LeaderboardService }
    ) {
      const receiveScoresDto = plainToClass(ReceiveScoresDto, {
        scores: args.scores,
      });
      const errors = await validate(receiveScoresDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      return await context.leaderboardService.processScores(
        receiveScoresDto.scores
      );
    },

    async manualTagUpdate(
      _: any,
      args: { discordID: string; newTagNumber: number },
      context: { leaderboardService: LeaderboardService }
    ) {
      const updateTagDto = plainToClass(UpdateTagDto, {
        discordID: args.discordID,
        tagNumber: args.newTagNumber,
      });
      const errors = await validate(updateTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      return await context.leaderboardService.updateTag(
        updateTagDto.discordID,
        updateTagDto.tagNumber
      );
    },

    async linkTag(
      _: any,
      args: { discordID: string; newTagNumber: number },
      context: { leaderboardService: LeaderboardService }
    ) {
      const linkTagDto = plainToClass(LinkTagDto, {
        discordID: args.discordID,
        newTagNumber: args.newTagNumber,
      });
      const errors = await validate(linkTagDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      return await context.leaderboardService.linkTag(
        linkTagDto.discordID,
        linkTagDto.newTagNumber
      );
    },
  },
};
