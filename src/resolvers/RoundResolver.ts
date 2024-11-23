import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { RoundService } from "../services/RoundService";
import { JoinRoundInput } from "../dto/join-round-input.dto";
import { ScheduleRoundInput } from "../dto/round-input.dto";
import { RoundState, Response } from "../types.generated";
import { EditRoundInput } from "../dto/edit-round-input.dto";

type ID = string; // Define ID as a string type

// Round Resolver in a Yoga-style GraphQL setup
export const RoundResolver = {
  Query: {
    // Fetch a list of rounds with optional pagination
    async getRounds(
      _: any,
      args: { limit?: number; offset?: number },
      context: { roundService: RoundService }
    ) {
      const { limit = 10, offset = 0 } = args;
      return await context.roundService.getRounds(limit, offset);
    },

    // Fetch details of a single round
    async getRound(
      _: any,
      args: { roundID: string },
      context: { roundService: RoundService }
    ) {
      return await context.roundService.getRound(args.roundID);
    },
  },

  Mutation: {
    // Schedule a new round
    async scheduleRound(
      _: any,
      args: { input: ScheduleRoundInput },
      context: { roundService: RoundService; discordID: string } // Assuming discordID is in context
    ) {
      const roundInput = {
        ...args.input,
        creatorID: context.discordID, // Set creatorID to discordID
      };

      const errors = await validate(roundInput);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      return await context.roundService.scheduleRound(roundInput);
    },

    async joinRound(
      _: any,
      args: { input: JoinRoundInput },
      context: {
        roundService: RoundService;
        discordID: string;
        leaderboardService: any;
      }
    ) {
      const { roundID, discordID, response } = args.input;

      if (!roundID) {
        throw new Error("roundID is required");
      }

      const round = await context.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.state !== RoundState.Upcoming) {
        throw new Error("You can only join rounds that are upcoming");
      }

      const tagNumber = await context.leaderboardService.getTagNumber(
        discordID
      );

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
    },

    async editRound(
      _: any,
      args: { roundID: ID; input: EditRoundInput },
      context: { roundService: RoundService; user: { id: ID } } // Assuming user ID is in context
    ) {
      const editRoundInput = plainToClass(EditRoundInput, args.input);
      const errors = await validate(editRoundInput);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      // Fetch the existing round to check the creator
      const existingRound = await context.roundService.getRound(args.roundID);

      // Check if the round exists
      if (!existingRound) {
        throw new Error("Round not found.");
      }

      // Check if the creatorID matches the user making the request
      if (existingRound.creatorID !== context.user.id) {
        throw new Error("You are not authorized to edit this round.");
      }

      return await context.roundService.editRound(args.roundID, editRoundInput);
    },

    // Submit a score for a round
    async submitScore(
      _: any,
      args: { roundID: string; score: number; tagNumber?: number }, // tagNumber is optional
      context: { roundService: RoundService; discordID: string }
    ) {
      const round = await context.roundService.getRound(args.roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      // Ensure the round is in progress
      if (round.state !== RoundState.InProgress) {
        throw new Error(
          "Scores can only be submitted for rounds that are in progress"
        );
      }

      // If tagNumber is not provided, set it to null
      const finalTagNumber = args.tagNumber ?? null;

      // Call the submitScore method with the final tag number
      return await context.roundService.submitScore(
        args.roundID,
        context.discordID,
        args.score,
        finalTagNumber // Pass either a number or null
      );
    },

    // Finalize the round and process scores
    async finalizeAndProcessScores(
      _: any,
      args: { roundID: string },
      context: { roundService: RoundService; scoreService: any }
    ) {
      const round = await context.roundService.getRound(args.roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.finalized) {
        throw new Error("Round has already been finalized");
      }

      // Process scores using the ScoreService
      await context.scoreService.processScores(round.roundID, round.scores);

      return await context.roundService.finalizeAndProcessScores(
        args.roundID,
        context.scoreService
      );
    },

    // Delete a round
    async deleteRound(
      _: any,
      args: { roundID: string },
      context: { roundService: RoundService; discordID: string }
    ) {
      return await context.roundService.deleteRound(
        args.roundID,
        context.discordID
      );
    },

    // Update a participant's response
    async updateParticipantResponse(
      _: any,
      args: { roundID: string; discordID: string; response: Response },
      context: { roundService: RoundService }
    ) {
      return await context.roundService.updateParticipantResponse(
        args.roundID,
        args.discordID,
        args.response
      );
    },
  },
};
