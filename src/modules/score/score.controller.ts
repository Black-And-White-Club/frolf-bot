// src/modules/score/score.controller.ts

import {
  Controller,
  Get,
  Put,
  Body,
  Param,
  HttpException,
  HttpStatus,
} from "@nestjs/common";
import { ScoreService } from "./score.service";
import { UpdateScoreDto } from "../../dto/score/update-score.dto";
import { validate } from "class-validator";
import { OnModuleInit } from "@nestjs/common";

// Import ConsumerService from the correct path
import { ConsumerService } from "src/rabbitmq/consumer"; // Make sure this is correct

@Controller("scores")
export class ScoreController {
  constructor(
    private readonly scoreService: ScoreService,
    private readonly consumerService: ConsumerService // Inject ConsumerService
  ) {}

  @Get(":roundId/:discordId")
  async getUserScore(
    @Param("discordId") discordId: string,
    @Param("roundId") roundId: string
  ) {
    try {
      const score = await this.scoreService.getUserScore(discordId, roundId);
      if (score === null) {
        throw new HttpException("Score not found", HttpStatus.NOT_FOUND);
      }
      return score;
    } catch (error) {
      console.error("Error fetching user score:", error);
      throw new HttpException(
        "Could not fetch user score",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  @Get(":roundId")
  async getScoresForRound(@Param("roundId") roundId: string) {
    try {
      return await this.scoreService.getScoresForRound(roundId);
    } catch (error) {
      console.error("Error fetching scores for round:", error);
      throw new HttpException(
        "Could not fetch scores for round",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  @Put(":roundId/:discordId")
  async updateScore(
    @Param("roundId") roundId: string,
    @Param("discordId") discordId: string,
    @Body() input: UpdateScoreDto
  ) {
    try {
      const errors = await validate(input);
      if (errors.length > 0) {
        throw new HttpException("Validation failed", HttpStatus.BAD_REQUEST);
      }

      return await this.scoreService.updateScore(
        roundId,
        discordId,
        input.score,
        input.tagNumber
      );
    } catch (error) {
      console.error("Error updating score:", error);
      throw new HttpException(
        "Could not update score",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  /**
   * Handler for processing scores from the `process_scores` queue.
   * This method will be triggered by incoming messages.
   */
  private async handleProcessScores(message: any) {
    try {
      const { roundID, scores } = message;
      console.log("Processing scores for round:", roundID);
      // Process the scores in the service
      await this.scoreService.processScores(roundID, scores);
    } catch (error) {
      console.error("Error processing scores:", error);
    }
  }
}
