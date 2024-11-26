// src/modules/score/score.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { scores as ScoreModel } from "../../schema";
import { eq, and } from "drizzle-orm";
import { Score as GraphQLScore } from "../../types.generated";
import { NodePgDatabase } from "drizzle-orm/node-postgres";

@Injectable()
export class ScoreService {
  constructor(@Inject("DATABASE_CONNECTION") private db: NodePgDatabase) {}

  async getUserScore(
    discordID: string,
    roundID: string
  ): Promise<GraphQLScore | null> {
    try {
      const scores = await this.db
        .select()
        .from(ScoreModel)
        .where(
          and(
            eq(ScoreModel.discordID, discordID),
            eq(ScoreModel.roundID, roundID)
          )
        );

      if (scores.length > 0) {
        const score = scores[0];
        return {
          ...score,
          tagNumber: score.tagNumber ?? null,
        };
      }

      return null;
    } catch (error) {
      console.error("Error fetching user score:", error);
      throw new Error("Failed to fetch user score");
    }
  }

  async getScoresForRound(roundID: string): Promise<GraphQLScore[]> {
    try {
      const scores = await this.db
        .select()
        .from(ScoreModel)
        .where(eq(ScoreModel.roundID, roundID));

      return scores.map((score) => ({
        ...score,
        tagNumber: score.tagNumber ?? null,
      }));
    } catch (error) {
      console.error("Error fetching scores for round:", error);
      throw new Error("Failed to fetch scores for round");
    }
  }

  async processScores(
    roundID: string,
    scores: { discordID: string; score: number; tagNumber?: number | null }[]
  ): Promise<GraphQLScore[]> {
    try {
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
          const newScore = await this.db
            .insert(ScoreModel)
            .values({
              discordID: scoreInput.discordID,
              roundID: roundID,
              score: scoreInput.score,
              tagNumber: scoreInput.tagNumber || null,
            })
            .returning()
            .execute();

          processedScores.push(newScore[0]);
        }
      }

      return processedScores;
    } catch (error) {
      console.error("Error processing scores:", error);
      throw new Error("Failed to process scores");
    }
  }

  async updateScore(
    roundID: string,
    discordID: string,
    score: number,
    tagNumber?: number | null
  ): Promise<GraphQLScore> {
    try {
      const existingScore = await this.getUserScore(discordID, roundID);
      if (!existingScore) {
        throw new Error("Score not found");
      }

      const updatedScore = await this.db
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
        .returning()
        .execute();

      return updatedScore[0];
    } catch (error) {
      console.error("Error updating score:", error);
      throw new Error("Failed to update score");
    }
  }
}
