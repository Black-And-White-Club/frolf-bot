import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { ScoreService } from "../services/ScoreService";
import { UpdateScoreDto } from "../dto/update-score.dto"; // DTO for updating scores
import { ProcessScoresDto } from "../dto/process-scores.dto"; // DTO for processing scores

export const ScoreResolver = {
  Query: {
    async getUserScore(
      _: any,
      args: { discordID: string; roundID: string },
      context: { scoreService: ScoreService }
    ) {
      const { discordID, roundID } = args;
      const score = await context.scoreService.getUserScore(discordID, roundID);
      if (score === null) {
        throw new Error(
          "Score not found for the provided discordID and roundID"
        );
      }
      return score;
    },

    async getScoresForRound(
      _: any,
      args: { roundID: string },
      context: { scoreService: ScoreService }
    ) {
      return await context.scoreService.getScoresForRound(args.roundID);
    },
  },
  Mutation: {
    async updateScore(
      _: any,
      args: { input: UpdateScoreDto },
      context: { scoreService: ScoreService }
    ) {
      const updateScoreDto = plainToClass(UpdateScoreDto, args.input);
      const errors = await validate(updateScoreDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      return await context.scoreService.updateScore(
        updateScoreDto.roundID,
        updateScoreDto.discordID,
        updateScoreDto.score,
        updateScoreDto.tagNumber
      );
    },

    async processScores(
      _: any,
      args: { input: ProcessScoresDto },
      context: { scoreService: ScoreService }
    ) {
      const processScoresDto = plainToClass(ProcessScoresDto, args.input);
      const errors = await validate(processScoresDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      // Call the service method with the roundID and the array of scores
      return await context.scoreService.processScores(
        processScoresDto.roundID,
        processScoresDto.scores
      );
    },
  },
};
