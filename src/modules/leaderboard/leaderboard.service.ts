// src/modules/leaderboard/leaderboard.service.ts

import { Inject, Injectable, HttpException, HttpStatus } from "@nestjs/common";
import { leaderboard as LeaderboardModel } from "./leaderboard.model";
import { eq } from "drizzle-orm";
import { NodePgDatabase } from "drizzle-orm/node-postgres";
import { Leaderboard } from "./leaderboard.entity";
import { QueueService } from "src/rabbitmq/queue.service";
import { UpdateTagSource } from "src/enums";

@Injectable()
export class LeaderboardService {
  constructor(
    @Inject("LEADERBOARD_DATABASE_CONNECTION")
    private db: NodePgDatabase,
    private readonly queueService: QueueService
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
              tagNumber: 0,
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
    const existingSwapId = await this.queueService.getActiveSwapId();

    let swapId = existingSwapId;
    if (!swapId) {
      swapId = `${Date.now()}-${Math.random().toString(36).substring(2, 15)}`;
    }

    const queueGroupName = `tag-swap-${swapId}`;

    try {
      console.log(
        `Manual tag swap initiated with ID: ${swapId} for discordID: ${discordID}, tagNumber: ${tagNumber}`
      );

      const result = await this.queueService.processTagSwapRequest(
        queueGroupName,
        UpdateTagSource.Manual
      );

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
        // Implement your notification logic here if needed
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
}
