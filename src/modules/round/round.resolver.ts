// src/modules/round/round.resolver.ts
import { Injectable } from "@nestjs/common";
import { RoundService } from "./round.service";
import { JoinRoundInput } from "../../dto/round/join-round-input.dto";
import { ScheduleRoundInput } from "../../dto/round/round-input.dto";
import { EditRoundInput } from "../../dto/round/edit-round-input.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { LeaderboardService } from "../leaderboard/leaderboard.service";

@Injectable()
export class RoundResolver {
  constructor(
    private readonly roundService: RoundService,
    private readonly leaderboardService: LeaderboardService
  ) {}

  // Manual resolver for getRounds
  async getRounds(limit?: number, offset?: number) {
    try {
      return await this.roundService.getRounds(limit, offset);
    } catch (error) {
      console.error("Error fetching rounds:", error);
      throw new Error("Failed to fetch rounds");
    }
  }

  // Manual resolver for getRound
  async getRound(roundID: string) {
    try {
      const round = await this.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }
      return round;
    } catch (error) {
      console.error("Error fetching round:", error);
      throw new Error("Failed to fetch round");
    }
  }

  // Manual resolver for scheduleRound
  async scheduleRound(_: any, { input }: { input: ScheduleRoundInput }) {
    try {
      console.log("Received input:", input); // Debugging line

      if (!input) {
        throw new Error("Input is undefined or null");
      }

      // Transform to the appropriate class instance (optional)
      const scheduleRoundInput = plainToClass(ScheduleRoundInput, input);

      // Validate the transformed input
      const errors = await validate(scheduleRoundInput);
      if (errors.length > 0) {
        throw new Error(
          "Validation failed: " + errors.map((e) => e.toString()).join(", ")
        );
      }

      // Call the roundService to schedule the round
      return await this.roundService.scheduleRound(scheduleRoundInput);
    } catch (error) {
      console.error("Error scheduling round:", error);
      throw new Error("Failed to schedule round");
    }
  }

  // Manual resolver for joinRound
  async joinRound(input: any) {
    try {
      // Manually transform the input to the JoinRoundInput class
      const joinRoundInput = plainToClass(JoinRoundInput, input);

      // Validate the transformed input using class-validator
      const errors = await validate(joinRoundInput);
      if (errors.length > 0) {
        throw new Error(
          "Validation failed: " + errors.map((e) => e.toString()).join(", ")
        );
      }

      // Now that input is validated, we can extract the properties
      const { roundID, discordID, response } = joinRoundInput;

      // Proceed with business logic
      const round = await this.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.state !== "UPCOMING") {
        throw new Error("You can only join rounds that are upcoming");
      }

      const existingParticipant = round.participants.find(
        (participant) => participant.discordID === discordID
      );
      if (existingParticipant) {
        throw new Error("You have already joined this round");
      }

      const tagNumber = await this.leaderboardService.getUserTag(discordID);

      // Call your roundService to join the round
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
    } catch (error) {
      console.error("Error joining round:", error);
      throw new Error("Failed to join round");
    }
  }

  // Manual resolver for editRound
  async editRound(roundID: string, input: any) {
    try {
      const editRoundInput = plainToClass(EditRoundInput, input);
      const errors = await validate(editRoundInput);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      const existingRound = await this.roundService.getRound(roundID);
      if (!existingRound) {
        throw new Error("Round not found.");
      }

      return await this.roundService.editRound(roundID, editRoundInput);
    } catch (error) {
      console.error("Error editing round:", error);
      throw new Error("Failed to edit round");
    }
  }

  // Manual resolver for submitScore
  async submitScore(
    roundID: string,
    discordID: string,
    score: number,
    tagNumber?: number
  ) {
    try {
      const round = await this.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.state !== "IN_PROGRESS") {
        throw new Error(
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
    } catch (error) {
      console.error("Error submitting score:", error);
      throw new Error("Failed to submit score");
    }
  }
}
