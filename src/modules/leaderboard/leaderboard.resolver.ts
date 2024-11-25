import { Resolver, Query, Mutation, Args, Context } from "@nestjs/graphql";
import { LeaderboardService } from "./leaderboard.service";
import { UpdateTagDto } from "../../dto/leaderboard/update-tag.dto"; // DTO for updating tags
import { ReceiveScoresDto } from "../../dto/leaderboard/receive-scores.dto"; // DTO for receiving scores
import { LinkTagDto } from "../../dto/leaderboard/link-tag.dto"; // DTO for linking tags
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

// Define the LeaderboardResolver class
@Resolver()
export class LeaderboardResolver {
  constructor(private readonly leaderboardService: LeaderboardService) {}

  @Query(() => [String]) // Adjust return type as necessary
  async getLeaderboard(
    @Args("page", { defaultValue: 1 }) page: number,
    @Args("limit", { defaultValue: 50 }) limit: number,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    return await context.leaderboardService.getLeaderboard(page, limit);
  }

  @Query(() => String) // Adjust return type as necessary
  async getUserTag(
    @Args("discordID") discordID: string,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    const tag = await context.leaderboardService.getUserTag(discordID);
    if (!tag) {
      throw new Error("Tag not found for the provided discordID");
    }
    return tag;
  }

  @Mutation(() => String) // Adjust return type as necessary
  async updateTag(
    @Args("discordID") discordID: string,
    @Args("tagNumber") tagNumber: number,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    const updateTagDto = plainToClass(UpdateTagDto, { discordID, tagNumber });
    const errors = await validate(updateTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await context.leaderboardService.updateTag(
      updateTagDto.discordID,
      updateTagDto.tagNumber
    );
  }

  @Mutation(() => String) // Adjust return type as necessary
  async receiveScores(
    @Args("scores")
    scores: Array<{ score: number; discordID: string; tagNumber?: number }>,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    const receiveScoresDto = plainToClass(ReceiveScoresDto, { scores });
    const errors = await validate(receiveScoresDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await context.leaderboardService.processScores(
      receiveScoresDto.scores
    );
  }

  @Mutation(() => String) // Adjust return type as necessary
  async manualTagUpdate(
    @Args("discordID") discordID: string,
    @Args("newTagNumber") newTagNumber: number,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    const updateTagDto = plainToClass(UpdateTagDto, {
      discordID,
      tagNumber: newTagNumber,
    });
    const errors = await validate(updateTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await context.leaderboardService.updateTag(
      updateTagDto.discordID,
      updateTagDto.tagNumber
    );
  }

  @Mutation(() => String) // Adjust return type as necessary
  async linkTag(
    @Args("discordID") discordID: string,
    @Args("newTagNumber") newTagNumber: number,
    @Context() context: { leaderboardService: LeaderboardService }
  ) {
    const linkTagDto = plainToClass(LinkTagDto, { discordID, newTagNumber });
    const errors = await validate(linkTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await context.leaderboardService.linkTag(
      linkTagDto.discordID,
      linkTagDto.newTagNumber
    );
  }
}
