// src/modules/leaderboard/leaderboard.service.ts

import { Inject, Injectable, HttpException, HttpStatus } from "@nestjs/common";
import { leaderboard as LeaderboardModel } from "./leaderboard.model";
import { eq, and, ne, sql } from "drizzle-orm";
import { NodePgDatabase } from "drizzle-orm/node-postgres";
import { Leaderboard } from "./leaderboard.entity";
import { QueueService } from "src/rabbitmq/queue.service";
import { UpdateTagSource } from "src/enums";
import { Publisher } from "src/rabbitmq/publisher";
import { RabbitSubscribe } from "@golevelup/nestjs-rabbitmq";

@Injectable()
export class LeaderboardService {
  constructor(
    @Inject("LEADERBOARD_DATABASE_CONNECTION")
    private db: NodePgDatabase,
    private readonly queueService: QueueService,
    private readonly publisher: Publisher
  ) {}

  async getLeaderboard(): Promise<Leaderboard> {
    try {
      const leaderboardEntry = await this.db
        .select()
        .from(LeaderboardModel)
        .where(eq(LeaderboardModel.active, true))
        .execute();

      if (leaderboardEntry.length > 0) {
        const leaderboard = leaderboardEntry[0];
        return {
          leaderboardData: leaderboard.leaderboardData || [],
        };
      } else {
        throw new HttpException(
          "No active leaderboard found",
          HttpStatus.NOT_FOUND
        );
      }
    } catch (error) {
      console.error("Error fetching leaderboard:", error);
      throw new HttpException(
        "Failed to fetch leaderboard",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  async getUserTag(
    discordID: string,
    tagNumber: number,
    source: UpdateTagSource
  ): Promise<{
    discordID: string;
    tagNumber: number;
    tagExists: boolean;
    message?: string;
  } | null> {
    try {
      const leaderboard = await this.getLeaderboard();
      const tagEntry = leaderboard.leaderboardData.find(
        (entry) => entry.discordID === discordID
      );

      switch (source) {
        case UpdateTagSource.Manual: {
          const tagExists = leaderboard.leaderboardData.some(
            (entry) =>
              entry.tagNumber === tagNumber && entry.discordID !== discordID
          );
          if (tagExists) {
            const swapId = await this.initiateManualTagSwap(
              discordID,
              tagNumber
            );
            return {
              discordID,
              tagNumber,
              tagExists: true,
              message: `Tag swap initiated with ID: ${swapId}`,
            };
          }
          break;
        }
        case UpdateTagSource.CreateUser: {
          const tagExists = leaderboard.leaderboardData.some(
            (entry) =>
              entry.tagNumber === tagNumber && entry.discordID !== discordID
          );
          if (tagExists) {
            return {
              discordID,
              tagNumber: 0, // Or handle the tag conflict differently
              tagExists: true,
              message: "Tag is taken, but account can be created without it.",
            };
          }
          break;
        }
      }

      return tagEntry ? { ...tagEntry, tagExists: false } : null;
    } catch (error) {
      console.error(
        `Error fetching user tag for discordID ${discordID}:`,
        error
      );
      throw new HttpException(
        `Could not fetch user tag for discordID ${discordID}`,
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  async initiateManualTagSwap(
    discordID: string,
    tagNumber: number
  ): Promise<string> {
    const swapId = `${Date.now()}-${Math.random()
      .toString(36)
      .substring(2, 15)}`;
    const queueGroupName = `tag-swap-${swapId}`;

    try {
      console.log(
        `Manual tag swap initiated with ID: ${swapId} for discordID: ${discordID}, tagNumber: ${tagNumber}`
      );

      await this.queueService.publishTagSwapRequest(queueGroupName, {
        discordID,
        tagNumber,
      });

      // Simulate waiting for a response (replace with actual logic)
      const result: {
        success: boolean;
        successfulSwaps?: { discordID: string; tagNumber: number }[];
        unmatchedUsers?: string[];
      } = await new Promise((resolve) => {
        setTimeout(() => {
          resolve({
            success: true,
            successfulSwaps: [{ discordID, tagNumber }],
          });
        }, 1000);
      });

      if (result.success) {
        if (result.successfulSwaps) {
          await this.updateTag(
            result.successfulSwaps,
            UpdateTagSource.ProcessScores
          );
        }

        console.log("Successful tag swaps:", result.successfulSwaps);
      } else {
        console.log(
          "Tag swap timed out. Unmatched users:",
          result.unmatchedUsers
        );
      }

      return swapId;
    } catch (error) {
      console.error("Error during tag swap:", error);
      throw new HttpException(
        "Tag swap failed",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  async updateTag(
    tagUpdates: { discordID: string; tagNumber: number }[],
    source: UpdateTagSource
  ): Promise<void> {
    try {
      if (source === UpdateTagSource.Manual) {
        return;
      }

      await this.db.transaction(async (tx) => {
        await tx
          .update(LeaderboardModel)
          .set({ active: false })
          .where(eq(LeaderboardModel.active, true))
          .execute();

        const currentLeaderboard = await this.getLeaderboard();
        let updatedLeaderboardData = currentLeaderboard.leaderboardData;

        tagUpdates.forEach((update) => {
          updatedLeaderboardData = updatedLeaderboardData.map((entry) =>
            entry.discordID === update.discordID
              ? { ...entry, tagNumber: update.tagNumber }
              : entry
          );
        });

        await tx
          .insert(LeaderboardModel)
          .values({ leaderboardData: updatedLeaderboardData, active: true })
          .execute();
      });

      console.log("Tags updated:", tagUpdates);
    } catch (error) {
      console.error("Error updating tag:", error);
      throw new HttpException(
        "Failed to update tag",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  async processScores(
    scores: { discordID: string; score: number; tagNumber?: number }[]
  ): Promise<void> {
    try {
      scores.sort((a, b) => {
        if (a.score === b.score) {
          return (a.tagNumber || 0) - (b.tagNumber || 0);
        }
        return a.score - b.score;
      });

      const leaderboardData = scores.map((score) => ({
        discordID: score.discordID,
        tagNumber: score.tagNumber || score.score,
      }));

      // Consider calling updateTag here to update the leaderboard with the processed scores
      // await this.updateTag(leaderboardData, UpdateTagSource.ProcessScores);
    } catch (error) {
      console.error("Error processing scores:", error);
      throw new HttpException(
        "Failed to process scores",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  // RabbitMQ Consumer for Check-Tag Messages
  @RabbitSubscribe({
    exchange: "main_exchange",
    routingKey: "check-tag",
    queue: "check-tag-queue",
  })
  async handleCheckTagMessage(message: any) {
    try {
      const { discordID, tagNumber } = message.content;
      console.log("Received check-tag message:", message.content);

      // Check if the tag already exists in the leaderboard
      const tagExists = await this.checkTagExists(discordID, tagNumber);

      // Publish the response with the correlation ID (access from message.properties)
      await this.publisher.publishMessage(
        "check-tag-responses",
        {
          discordID,
          tagExists,
        },
        { correlationId: message.properties.correlationId }
      );

      console.log("Published tag existence response:", {
        discordID,
        tagExists,
      });
    } catch (error) {
      console.error("Error processing check-tag message:", error);
    }
  }

  private async checkTagExists(
    discordID: string,
    tagNumber: number
  ): Promise<boolean> {
    const leaderboard = await this.getLeaderboard();
    return leaderboard.leaderboardData.some(
      (entry) => entry.tagNumber === tagNumber && entry.discordID !== discordID
    );
  }
}
