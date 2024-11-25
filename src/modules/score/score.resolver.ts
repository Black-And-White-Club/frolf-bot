import { Resolver, Query, Mutation, Args, Context } from "@nestjs/graphql";
import { ScoreService } from "./score.service";
import { UpdateScoreDto } from "../../dto/score/update-score.dto"; // DTO for updating scores
import { ProcessScoresDto } from "../../dto/score/process-scores.dto"; // DTO for processing scores
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { GraphQLResolveInfo } from "graphql";

@Resolver()
export class ScoreResolver {
  constructor(private readonly scoreService: ScoreService) {}

  @Query(() => String) // Adjust the return type as needed
  async getUserScore(
    @Args("discordID") discordID: string,
    @Args("roundID") roundID: string,
    @Context() context: { scoreService: ScoreService },
    info?: GraphQLResolveInfo
  ): Promise<any> {
    const score = await this.scoreService.getUserScore(discordID, roundID);
    if (score === null) {
      throw new Error("Score not found for the provided discordID and roundID");
    }
    return score;
  }

  @Query(() => [String]) // Adjust the return type as needed
  async getScoresForRound(
    @Args("roundID") roundID: string,
    @Context() context: { scoreService: ScoreService },
    info?: GraphQLResolveInfo
  ): Promise<any[]> {
    return await this.scoreService.getScoresForRound(roundID);
  }

  @Mutation(() => String) // Adjust the return type as needed
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

  @Mutation(() => String) // Adjust the return type as needed
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
        score: parseInt(score.score.toString(), 10), // Ensure score is an integer
      }))
    );
  }
}
