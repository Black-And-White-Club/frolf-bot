// round.resolver.ts
import { Resolver, Query, Mutation, Args } from "@nestjs/graphql";
import { RoundService } from "./round.service";
import { JoinRoundInput } from "../../dto/round/join-round-input.dto";
import { ScheduleRoundInput } from "../../dto/round/round-input.dto";
import { EditRoundInput } from "../../dto/round/edit-round-input.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { GraphQLError } from "graphql";
import { LeaderboardService } from "../leaderboard/leaderboard.service";

@Resolver()
export class RoundResolver {
  constructor(
    private readonly roundService: RoundService,
    private readonly leaderboardService: LeaderboardService
  ) {
    console.log("RoundResolver roundService:", roundService); // Add this line
  }

  @Query(() => [String])
  async getRounds(
    @Args("limit", { nullable: true }) limit?: number,
    @Args("offset", { nullable: true }) offset?: number
  ) {
    return await this.roundService.getRounds(limit, offset);
  }

  @Query(() => String)
  async getRound(@Args("roundID") roundID: string) {
    const round = await this.roundService.getRound(roundID);
    if (!round) {
      throw new GraphQLError("Round not found");
    }
    return round;
  }

  @Mutation(() => String)
  async scheduleRound(@Args("input") input: ScheduleRoundInput) {
    // No validation here
    return await this.roundService.scheduleRound(input);
  }

  @Mutation(() => String)
  async joinRound(@Args("input") input: JoinRoundInput) {
    const { roundID, discordID, response } = input;

    const round = await this.roundService.getRound(roundID);
    if (!round) {
      throw new GraphQLError("Round not found");
    }

    if (round.state !== "UPCOMING") {
      throw new GraphQLError("You can only join rounds that are upcoming");
    }

    const existingParticipant = round.participants.find(
      (participant) => participant.discordID === discordID
    );
    if (existingParticipant) {
      throw new GraphQLError("You have already joined this round");
    }

    const tagNumber = await this.leaderboardService.getUserTag(discordID);

    await this.roundService.joinRound({
      roundID,
      discordID,
      response,
      tagNumber: tagNumber?.tagNumber || null,
    });

    return {
      roundID,
      discordID,
      response,
    };
  }

  @Mutation(() => String)
  async editRound(
    @Args("roundID") roundID: string,
    @Args("input") input: EditRoundInput
  ) {
    const editRoundInput = plainToClass(EditRoundInput, input);
    const errors = await validate(editRoundInput);
    if (errors.length > 0) {
      throw new GraphQLError("Validation failed!");
    }

    const existingRound = await this.roundService.getRound(roundID);
    if (!existingRound) {
      throw new GraphQLError("Round not found.");
    }

    return await this.roundService.editRound(roundID, editRoundInput);
  }

  @Mutation(() => String)
  async submitScore(
    @Args("roundID") roundID: string,
    @Args("discordID") discordID: string,
    @Args("score") score: number,
    @Args("tagNumber", { nullable: true }) tagNumber?: number
  ) {
    const round = await this.roundService.getRound(roundID);
    if (!round) {
      throw new GraphQLError("Round not found");
    }

    if (round.state !== "IN_PROGRESS") {
      throw new GraphQLError(
        "Scores can only be submitted for rounds that are in progress"
      );
    }

    const finalTagNumber = tagNumber ?? null;
    return await this.roundService.submitScore(
      roundID,
      discordID,
      score,
      finalTagNumber
    );
  }
}
