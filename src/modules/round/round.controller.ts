// src/modules/round/round.controller.ts

import { Controller, Get, Post, Body, Param, Put, Query } from "@nestjs/common";
import { RoundService } from "./round.service";
import { JoinRoundInput } from "src/dto/round/join-round-input.dto";
import { ScheduleRoundInput } from "src/dto/round/round-input.dto";
import { EditRoundInput } from "src/dto/round/edit-round-input.dto";
import { SubmitScoreDto } from "src/dto/round/submit-score.dto";
import { Publisher } from "src/rabbitmq/publisher";

@Controller("rounds")
export class RoundController {
  constructor(
    private readonly roundService: RoundService,
    private readonly publisher: Publisher
  ) {}

  @Get()
  async getRounds(
    @Query("limit") limit?: number,
    @Query("offset") offset?: number
  ) {
    try {
      return await this.roundService.getRounds(limit, offset);
    } catch (error) {
      console.error("Error fetching rounds:", error);
      throw error;
    }
  }

  @Get(":roundID")
  async getRound(@Param("roundID") roundID: string) {
    try {
      const round = await this.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }
      return round;
    } catch (error) {
      console.error("Error fetching round:", error);
      throw error;
    }
  }

  @Post()
  async scheduleRound(@Body() input: ScheduleRoundInput) {
    try {
      return await this.roundService.scheduleRound(input);
    } catch (error) {
      console.error("Error scheduling round:", error);
      throw error;
    }
  }

  @Post("join")
  async joinRound(@Body() input: JoinRoundInput) {
    try {
      const { roundID, discordID, response } = input;

      // Get the round details
      const round = await this.roundService.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      // Check if the round is in the correct state
      if (round.state !== "UPCOMING") {
        throw new Error("You can only join rounds that are upcoming");
      }

      // Check if the user is already a participant
      const existingParticipant = round.participants.find(
        (participant) => participant.discordID === discordID
      );
      if (existingParticipant) {
        throw new Error("You have already joined this round");
      }

      // Fetch the tag number using the publisher
      const tagNumber = await this.publisher.publishAndGetResponse(
        "get-tag", // exchange
        "round_responses", // routing key
        { discordID } // message
      );

      // Ensure tagNumber is a valid number or null
      const validTagNumber = typeof tagNumber === "number" ? tagNumber : null;

      // Join the round
      await this.roundService.joinRound({
        roundID,
        discordID,
        response,
        tagNumber: validTagNumber, // Safely handle tagNumber being null
      });

      return {
        roundID,
        discordID,
        response,
      };
    } catch (error) {
      console.error("Error joining round:", error);
      throw error;
    }
  }

  @Put(":roundID")
  async editRound(
    @Param("roundID") roundID: string,
    @Body() input: EditRoundInput
  ) {
    try {
      const existingRound = await this.roundService.getRound(roundID);
      if (!existingRound) {
        throw new Error("Round not found.");
      }

      return await this.roundService.editRound(roundID, input);
    } catch (error) {
      console.error("Error editing round:", error);
      throw error;
    }
  }

  @Post(":roundID/scores")
  async submitScore(
    @Param("roundID") roundID: string,
    @Body() input: SubmitScoreDto
  ) {
    try {
      const { discordID, score, tagNumber } = input;

      const round = await this.roundService.submitScore(
        roundID,
        discordID,
        score,
        tagNumber ?? null // Use nullish coalescing operator
      );

      // Publish the updated scores to RabbitMQ for processing
      await this.publisher.publishMessage("process_scores", {
        roundID,
        scores: round.scores,
      });

      return { message: "Score submitted successfully" };
    } catch (error) {
      console.error("Error submitting score:", error);
      throw new Error("Failed to submit score");
    }
  }

  @Post(":roundID/finalize")
  async finalizeRound(@Param("roundID") roundID: string) {
    try {
      await this.roundService.finalizeAndProcessScores(roundID);
      return { message: "Round finalized successfully" };
    } catch (error) {
      console.error("Error finalizing round:", error);
      throw new Error("Failed to finalize round");
    }
  }
}
