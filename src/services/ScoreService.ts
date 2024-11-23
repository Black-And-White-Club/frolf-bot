import { Injectable, Inject } from "@nestjs/common";
import { scores as ScoreModel } from "../db/models/Score"; // Ensure this import is correct
import { eq, and } from "drizzle-orm"; // Import and for combining conditions
import { Score as GraphQLScore } from "../types.generated"; // Importing the GraphQL types
import { drizzle } from "drizzle-orm/node-postgres";

@Injectable()
export class ScoreService {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>
  ) {
    console.log("Injected DB instance:", db);
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
        tagNumber: score.tagNumber || null, // Ensure this is number | null
      };
    }

    return null; // No score found
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
      tagNumber: score.tagNumber || null, // Ensure this is number | null
    }));
  }

  async processScores(
    roundID: string,
    scores: { discordID: string; score: number; tagNumber?: number | null }[]
  ): Promise<GraphQLScore[]> {
    const processedScores: GraphQLScore[] = [];

    for (const scoreInput of scores) {
      // Check for existing score for the given discordID and roundID
      const existingScore = await this.getUserScore(
        scoreInput.discordID,
        roundID
      );

      if (existingScore) {
        // If an existing score is found, throw an error
        throw new Error(
          "Score for this Discord ID already exists for the given round"
        );
      } else {
        // Create new score if it doesn't exist
        await this.db
          .insert(ScoreModel)
          .values({
            discordID: scoreInput.discordID,
            roundID: roundID,
            score: scoreInput.score,
            tagNumber: scoreInput.tagNumber || null, // Ensure this is number | null
          })
          .execute();

        // Push processed score to return array
        processedScores.push({
          __typename: "Score",
          discordID: scoreInput.discordID,
          score: scoreInput.score,
          tagNumber: scoreInput.tagNumber || null, // This should now be valid
        });
      }
    }

    return processedScores;
  }

  async updateScore(
    discordID: string,
    roundID: string,
    score: number,
    tagNumber?: number | null // Allow tagNumber to be optional and null
  ): Promise<GraphQLScore> {
    const existingScore = await this.getUserScore(discordID, roundID);
    if (!existingScore) {
      throw new Error("Score not found");
    }

    await this.db
      .update(ScoreModel)
      .set({
        score: score,
        tagNumber: tagNumber || null, // Ensure this is number | null
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
      tagNumber: tagNumber || null, // Ensure this is number | null
    };
  }
}
