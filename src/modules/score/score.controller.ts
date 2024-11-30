// src/modules/score/score.controller.ts

import {
  Controller,
  Get,
  Post,
  Body,
  Param,
  Put,
  HttpException,
  HttpStatus,
} from "@nestjs/common";
import { ScoreService } from "./score.service";
import { UpdateScoreDto } from "../../dto/score/update-score.dto";
import { Consumer } from "src/rabbitmq/consumer";
import { validate } from "class-validator";

@Controller("scores")
export class ScoreController {
  constructor(
    private readonly scoreService: ScoreService,
    private readonly consumer: Consumer
  ) {
    this.consumer.consumeMessages(
      "process_scores",
      this.handleProcessScores,
      "processScoresConsumer"
    );
  }

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

  private async handleProcessScores(message: any) {
    try {
      const { roundID, scores } = message;
      console.log("Processing scores for round:", roundID);
      await this.scoreService.processScores(roundID, scores);
    } catch (error) {
      console.error("Error processing scores:", error);
    }
  }
}
