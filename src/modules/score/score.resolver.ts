// src/modules/score/score.resolver.ts
import { Injectable } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { UpdateScoreDto } from "../../dto/score/update-score.dto";
import { ProcessScoresDto } from "../../dto/score/process-scores.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Injectable()
export class ScoreResolver {
  constructor(private readonly scoreService: ScoreService) {}

  async getUserScore(discordID: string, roundID: string): Promise<any> {
    try {
      const score = await this.scoreService.getUserScore(discordID, roundID);
      if (score === null) {
        throw new Error(
          "Score not found for the provided discordID and roundID"
        );
      }
      return score;
    } catch (error) {
      console.error("Error fetching user score:", error);
      if (error instanceof Error) {
        throw new Error(`Could not fetch user score: ${error.message}`);
      } else {
        throw new Error(`Could not fetch user score: ${error}`);
      }
    }
  }

  async getScoresForRound(roundID: string): Promise<any[]> {
    try {
      return await this.scoreService.getScoresForRound(roundID);
    } catch (error) {
      console.error("Error fetching scores for round:", error);
      if (error instanceof Error) {
        throw new Error(`Could not fetch scores for round: ${error.message}`);
      } else {
        throw new Error(`Could not fetch scores for round: ${error}`);
      }
    }
  }

  async updateScore(input: UpdateScoreDto): Promise<any> {
    try {
      const updateScoreDto = plainToClass(UpdateScoreDto, input);
      const errors = await validate(updateScoreDto);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }
      return await this.scoreService.updateScore(
        updateScoreDto.roundID,
        updateScoreDto.discordID,
        updateScoreDto.score,
        updateScoreDto.tagNumber
      );
    } catch (error) {
      console.error("Error updating score:", error);
      if (error instanceof Error) {
        throw new Error(`Could not update score: ${error.message}`);
      } else {
        throw new Error(`Could not update score: ${error}`);
      }
    }
  }

  async processScores(input: ProcessScoresDto): Promise<any> {
    try {
      const processScoresDto = plainToClass(ProcessScoresDto, input);
      const errors = await validate(processScoresDto);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }
      return await this.scoreService.processScores(
        processScoresDto.roundID,
        processScoresDto.scores.map((score) => ({
          ...score,
          score: parseInt(score.score.toString(), 10),
        }))
      );
    } catch (error) {
      console.error("Error processing scores:", error);
      if (error instanceof Error) {
        throw new Error(`Could not process scores: ${error.message}`);
      } else {
        throw new Error(`Could not process scores: ${error}`);
      }
    }
  }
}
