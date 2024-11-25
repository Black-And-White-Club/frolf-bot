// leaderboard.resolver.ts
import { Resolver, Query, Mutation, Args } from "@nestjs/graphql";
import { LeaderboardService } from "./leaderboard.service";
import { UpdateTagDto } from "../../dto/leaderboard/update-tag.dto";
import { ReceiveScoresDto } from "../../dto/leaderboard/receive-scores.dto";
import { LinkTagDto } from "../../dto/leaderboard/link-tag.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Resolver()
export class LeaderboardResolver {
  constructor(private readonly leaderboardService: LeaderboardService) {}

  @Query(() => [String])
  async getLeaderboard(
    @Args("page", { defaultValue: 1 }) page: number,
    @Args("limit", { defaultValue: 50 }) limit: number
  ) {
    return await this.leaderboardService.getLeaderboard(page, limit);
  }

  @Query(() => String)
  async getUserTag(@Args("discordID") discordID: string) {
    const tag = await this.leaderboardService.getUserTag(discordID);
    if (!tag) {
      throw new Error("Tag not found for the provided discordID");
    }
    return tag;
  }

  @Mutation(() => String)
  async updateTag(
    @Args("discordID") discordID: string,
    @Args("tagNumber") tagNumber: number
  ) {
    const updateTagDto = plainToClass(UpdateTagDto, { discordID, tagNumber });
    const errors = await validate(updateTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await this.leaderboardService.updateTag(
      updateTagDto.discordID,
      updateTagDto.tagNumber
    );
  }

  @Mutation(() => String)
  async receiveScores(
    @Args("scores")
    scores: Array<{ score: number; discordID: string; tagNumber?: number }>
  ) {
    const receiveScoresDto = plainToClass(ReceiveScoresDto, { scores });
    const errors = await validate(receiveScoresDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await this.leaderboardService.processScores(receiveScoresDto.scores);
  }

  @Mutation(() => String)
  async manualTagUpdate(
    @Args("discordID") discordID: string,
    @Args("newTagNumber") newTagNumber: number
  ) {
    const updateTagDto = plainToClass(UpdateTagDto, {
      discordID,
      tagNumber: newTagNumber,
    });
    const errors = await validate(updateTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await this.leaderboardService.updateTag(
      updateTagDto.discordID,
      updateTagDto.tagNumber
    );
  }

  @Mutation(() => String)
  async linkTag(
    @Args("discordID") discordID: string,
    @Args("newTagNumber") newTagNumber: number
  ) {
    const linkTagDto = plainToClass(LinkTagDto, { discordID, newTagNumber });
    const errors = await validate(linkTagDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }
    return await this.leaderboardService.linkTag(
      linkTagDto.discordID,
      linkTagDto.newTagNumber
    );
  }
}
