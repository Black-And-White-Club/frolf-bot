// score.resolver.ts
import { Resolver, Query, Args, Mutation } from "@nestjs/graphql";
import { ScoreService } from "./score.service";
import { UpdateScoreDto } from "../../dto/score/update-score.dto";
import { ProcessScoresDto } from "../../dto/score/process-scores.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Resolver()
export class ScoreResolver {
  constructor(private readonly scoreService: ScoreService) {}

  @Query(() => String)
  async getUserScore(
    @Args("discordID") discordID: string,
    @Args("roundID") roundID: string
  ): Promise<any> {
    const score = await this.scoreService.getUserScore(discordID, roundID);
    if (score === null) {
      throw new Error("Score not found for the provided discordID and roundID");
    }
    return score;
  }

  @Query(() => [String])
  async getScoresForRound(@Args("roundID") roundID: string): Promise<any[]> {
    return await this.scoreService.getScoresForRound(roundID);
  }

  @Mutation(() => String)
  async updateScore(@Args("input") input: UpdateScoreDto): Promise<any> {
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
  }

  @Mutation(() => String)
  async processScores(@Args("input") input: ProcessScoresDto): Promise<any> {
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
  }
}
