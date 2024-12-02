// src/modules/score/score.service.ts

import { Inject, Injectable } from "@nestjs/common";
import { scores as ScoreModel } from "./score.model";
import { eq, and } from "drizzle-orm";
import { NodePgDatabase } from "drizzle-orm/node-postgres";
import { Score } from "./score.entity";
import { Publisher } from "src/rabbitmq/publisher";
import { ConsumerService } from "src/rabbitmq/consumer";
import { RabbitSubscribe } from "@golevelup/nestjs-rabbitmq";

@Injectable()
export class ScoreService {
  constructor(
    @Inject("SCORE_DATABASE_CONNECTION") private db: NodePgDatabase,
    private readonly publisher: Publisher,
    private readonly consumerService: ConsumerService
  ) {}

  async getUserScore(
    discordID: string,
    roundID: string
  ): Promise<Score | null> {
    try {
      const scoreData = await this.db
        .select()
        .from(ScoreModel)
        .where(
          and(
            eq(ScoreModel.discordID, discordID),
            eq(ScoreModel.roundID, roundID)
          )
        )
        .execute();

      if (scoreData.length > 0) {
        return this.mapScoreToEntity(scoreData[0]);
      }

      return null;
    } catch (error) {
      console.error("Error fetching user score:", error);
      throw new Error("Failed to fetch user score");
    }
  }

  async getScoresForRound(roundID: string): Promise<Score[]> {
    try {
      const scoreData = await this.db
        .select()
        .from(ScoreModel)
        .where(eq(ScoreModel.roundID, roundID));

      return scoreData.map((score) => this.mapScoreToEntity(score));
    } catch (error) {
      console.error("Error fetching scores for round:", error);
      throw new Error("Failed to fetch scores for round");
    }
  }

  async processScores(
    roundID: string,
    scores:
      | { discordID: string; score: number; tagNumber?: number | null }[]
      | { discordID: string; score: number; tagNumber?: number | null }
  ): Promise<void> {
    try {
      const scoresArray = Array.isArray(scores) ? scores : [scores];

      await this.db.transaction(async (tx) => {
        const processedScores = [];

        for (const scoreInput of scoresArray) {
          const existingScore = await this.getUserScore(
            scoreInput.discordID,
            roundID
          );
          if (existingScore) {
            await tx
              .update(ScoreModel)
              .set({
                score: scoreInput.score,
                tagNumber: scoreInput.tagNumber || null,
              })
              .where(
                and(
                  eq(ScoreModel.discordID, scoreInput.discordID),
                  eq(ScoreModel.roundID, roundID)
                )
              );
          } else {
            await tx.insert(ScoreModel).values({
              discordID: scoreInput.discordID,
              roundID: roundID,
              score: scoreInput.score,
              tagNumber: scoreInput.tagNumber || null,
            });
          }

          processedScores.push({
            discordID: scoreInput.discordID,
            score: scoreInput.score,
            tagNumber: scoreInput.tagNumber,
          });
        }

        const scoresForLeaderboard = processedScores.filter(
          (score) => score.tagNumber !== null && score.tagNumber !== undefined
        );

        await this.publisher.publishMessage("update_leaderboard", {
          roundID,
          scores: scoresForLeaderboard,
        });
      });
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
  ): Promise<Score> {
    try {
      const existingScore = await this.getUserScore(discordID, roundID);
      if (!existingScore) {
        throw new Error("Score not found");
      }

      const updatedScoreData = await this.db
        .update(ScoreModel)
        .set({ score, tagNumber: tagNumber || null })
        .where(
          and(
            eq(ScoreModel.discordID, discordID),
            eq(ScoreModel.roundID, roundID)
          )
        )
        .returning()
        .execute();

      await this.publisher.publishMessage("update_leaderboard", {
        roundID,
      });

      return this.mapScoreToEntity(updatedScoreData[0]);
    } catch (error) {
      console.error("Error updating score:", error);
      throw new Error("Failed to update score");
    }
  }

  private mapScoreToEntity(scoreData: any): Score {
    const score = new Score();
    score.discordID = scoreData.discordID;
    score.roundID = scoreData.roundID;
    score.score = scoreData.score;
    score.tagNumber = scoreData.tagNumber;
    score.createdAt = scoreData.createdAt;
    score.updatedAt = scoreData.updatedAt;
    return score;
  }

  @RabbitSubscribe({
    exchange: "main_exchange",
    routingKey: "process_scores",
    queue: "process_scores",
  })
  async handleScoreMessage(message: any) {
    return this.consumerService.handleIncomingMessage(
      message,
      "process_scores",
      "process_scores"
    );
  }
}
