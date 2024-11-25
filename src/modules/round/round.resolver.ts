import { Resolver, Query, Mutation, Args, Context } from "@nestjs/graphql";
import { RoundService } from "./round.service";
import { JoinRoundInput } from "../../dto/round/join-round-input.dto";
import { ScheduleRoundInput } from "../../dto/round/round-input.dto";
import { RoundState, Response } from "../../types.generated";
import { EditRoundInput } from "../../dto/round/edit-round-input.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Resolver()
export class RoundResolver {
  constructor(
    private readonly roundService: RoundService,
    // Assuming leaderboardService is also injected
    private readonly leaderboardService: any
  ) {}

  @Query(() => [String]) // Adjust return type as necessary
  async getRounds(
    @Args("limit") limit: number = 10,
    @Args("offset") offset: number = 0,
    @Context() context: { roundService: RoundService }
  ) {
    return await context.roundService.getRounds(limit, offset);
  }

  @Query(() => String) // Adjust return type as necessary
  async getRound(
    @Args("roundID") roundID: string,
    @Context() context: { roundService: RoundService }
  ) {
    return await context.roundService.getRound(roundID);
  }

  @Mutation(() => String) // Adjust return type as necessary
  async scheduleRound(
    @Args("input") input: ScheduleRoundInput,
    @Context() context: { roundService: RoundService; discordID: string }
  ) {
    const roundInput = {
      ...input,
      creatorID: context.discordID, // Set creatorID to discordID
    };

    const errors = await validate(roundInput);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }

    return await context.roundService.scheduleRound(roundInput);
  }

  @Mutation(() => String) // Adjust return type as necessary
  async joinRound(
    @Args("input") input: JoinRoundInput,
    @Context() context: { roundService: RoundService; discordID: string }
  ) {
    const { roundID, discordID, response } = input;

    if (!roundID) {
      throw new Error("roundID is required");
    }

    const round = await context.roundService.getRound(roundID);
    if (!round) {
      throw new Error("Round not found");
    }

    if (round.state !== "UPCOMING") {
      throw new Error("You can only join rounds that are upcoming");
    }

    // Check if the user has already joined the round
    const existingParticipant = round.participants.find(
      (participant) => participant.discordID === discordID
    );
    if (existingParticipant) {
      throw new Error("You have already joined this round");
    }

    const tagNumber = await this.leaderboardService.getTagNumber(discordID);

    // Add the participant to the round
    await context.roundService.joinRound({
      roundID,
      discordID,
      response,
      tagNumber: tagNumber || null,
    });

    return {
      roundID,
      discordID,
      response,
    };
  }
  @Mutation(() => String) // Adjust return type as necessary
  async editRound(
    @Args("roundID") roundID: string,
    @Args("input") input: EditRoundInput,
    @Context() context: { roundService: RoundService; user: { id: string } }
  ) {
    const editRoundInput = plainToClass(EditRoundInput, input);
    const errors = await validate(editRoundInput);
    if (errors.length > 0) {
      throw new Error("Validation failed!");
    }

    // Fetch the existing round to check the creator
    const existingRound = await context.roundService.getRound(roundID);

    // Check if the round exists
    if (!existingRound) {
      throw new Error("Round not found.");
    }

    // Check if the creatorID matches the user making the request
    if (existingRound.creatorID !== context.user.id) {
      throw new Error("You are not authorized to edit this round.");
    }

    return await context.roundService.editRound(roundID, editRoundInput);
  }

  @Mutation(() => String) // Adjust return type as necessary
  async submitScore(
    @Context() context: { roundService: RoundService; discordID: string },
    @Args("roundID") roundID: string,
    @Args("score") score: number,
    @Args("tagNumber") tagNumber?: number
  ) {
    const round = await context.roundService.getRound(roundID);
    if (!round) {
      throw new Error("Round not found");
    }

    // Ensure the round is in progress
    if (round.state !== "IN_PROGRESS") {
      throw new Error(
        "Scores can only be submitted for rounds that are in progress"
      );
    }

    // If tagNumber is not provided, set it to null
    const finalTagNumber = tagNumber ?? null;

    // Call the submitScore method with the final tag number
    return await context.roundService.submitScore(
      roundID,
      context.discordID,
      score,
      finalTagNumber // Pass either a number or null
    );
  }

  @Mutation(() => String) // Adjust return type as necessary
  async finalizeAndProcessScores(
    @Args("roundID") roundID: string,
    @Context() context: { roundService: RoundService; scoreService: any }
  ) {
    const round = await context.roundService.getRound(roundID);
    if (!round) {
      throw new Error("Round not found");
    }

    if (round.finalized) {
      throw new Error("Round has already been finalized");
    }

    // Process scores using the ScoreService
    await context.scoreService.processScores(round.roundID, round.scores);

    return await context.roundService.finalizeAndProcessScores(
      roundID,
      context.scoreService
    );
  }

  @Mutation(() => String) // Adjust return type as necessary
  async deleteRound(
    @Args("roundID") roundID: string,
    @Context() context: { roundService: RoundService; discordID: string }
  ) {
    return await context.roundService.deleteRound(roundID, context.discordID);
  }

  @Mutation(() => String) // Adjust return type as necessary
  async updateParticipantResponse(
    @Args("roundID") roundID: string,
    @Args("discordID") discordID: string,
    @Args("response") response: Response,
    @Context() context: { roundService: RoundService }
  ) {
    return await context.roundService.updateParticipantResponse(
      roundID,
      discordID,
      response
    );
  }
}
