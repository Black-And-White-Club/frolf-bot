// score.service.ts
import { Injectable, Inject } from "@nestjs/common";
import { scores as ScoreModel } from "./score.model";
import { eq, and } from "drizzle-orm";
import { Score as GraphQLScore } from "../../types.generated";
import { drizzle } from "drizzle-orm/node-postgres";

@Injectable()
export class ScoreService {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>
  ) {
    console.log("Injected DB instance:", db);
    console.log("ScoreService db:", this.db);
  }

  async getUserScore(
    discordID: string,
    roundID: string
  ): Promise<GraphQLScore | null> {
    const scores = await this.db
      .select()
      .from(ScoreModel)
      .where(
        and(
          eq(ScoreModel.discordID, discordID),
          eq(ScoreModel.roundID, roundID)
        )
      )
      .execute();

    if (scores.length > 0) {
      const score = scores[0];
      return {
        __typename: "Score",
        discordID: score.discordID,
        score: score.score,
        tagNumber: score.tagNumber || null,
      };
    }

    return null;
  }

  async getScoresForRound(roundID: string): Promise<GraphQLScore[]> {
    const scores = await this.db
      .select()
      .from(ScoreModel)
      .where(eq(ScoreModel.roundID, roundID))
      .execute();

    return scores.map((score) => ({
      __typename: "Score",
      discordID: score.discordID,
      score: score.score,
      tagNumber: score.tagNumber || null,
    }));
  }

  async processScores(
    roundID: string,
    scores: { discordID: string; score: number; tagNumber?: number | null }[]
  ): Promise<GraphQLScore[]> {
    const processedScores: GraphQLScore[] = [];

    for (const scoreInput of scores) {
      const existingScore = await this.getUserScore(
        scoreInput.discordID,
        roundID
      );

      if (existingScore) {
        throw new Error(
          "Score for this Discord ID already exists for the given round"
        );
      } else {
        await this.db
          .insert(ScoreModel)
          .values({
            discordID: scoreInput.discordID,
            roundID: roundID,
            score: scoreInput.score,
            tagNumber: scoreInput.tagNumber || null,
          })
          .execute();

        processedScores.push({
          __typename: "Score",
          discordID: scoreInput.discordID,
          score: scoreInput.score,
          tagNumber: scoreInput.tagNumber || null,
        });
      }
    }

    return processedScores;
  }

  async updateScore(
    discordID: string,
    roundID: string,
    score: number,
    tagNumber?: number | null
  ): Promise<GraphQLScore> {
    const existingScore = await this.getUserScore(discordID, roundID);
    if (!existingScore) {
      throw new Error("Score not found");
    }

    await this.db
      .update(ScoreModel)
      .set({
        score: score,
        tagNumber: tagNumber || null,
      })
      .where(
        and(
          eq(ScoreModel.discordID, discordID),
          eq(ScoreModel.roundID, roundID)
        )
      )
      .execute();

    return {
      __typename: "Score",
      discordID: discordID,
      score: score,
      tagNumber: tagNumber || null,
    };
  }
}
