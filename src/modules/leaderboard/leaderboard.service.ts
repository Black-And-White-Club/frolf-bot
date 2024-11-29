// src/modules/leaderboard/leaderboard.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { leaderboard as LeaderboardModel } from "src/schema";
import { asc, eq } from "drizzle-orm";
import { NodePgDatabase } from "drizzle-orm/node-postgres";

@Injectable()
export class LeaderboardService {
  constructor(
    @Inject("LEADERBOARD_DATABASE_CONNECTION") private db: NodePgDatabase
  ) {}

  async getLeaderboard(): Promise<{
    leaderboardData: Array<{ discordID: string; tagNumber: number }>;
  }> {
    try {
      const leaderboardEntry = await this.db
        .select()
        .from(LeaderboardModel)
        .orderBy(asc(LeaderboardModel.leaderboardID))
        .limit(1)
        .execute();

      if (leaderboardEntry.length > 0) {
        const leaderboard = leaderboardEntry[0];
        return {
          leaderboardData: leaderboard.leaderboardData || [],
        };
      } else {
        const initialLeaderboard = { leaderboardData: [] };
        await this.db
          .insert(LeaderboardModel)
          .values(initialLeaderboard)
          .execute();
        return initialLeaderboard;
      }
    } catch (error) {
      console.error("Error fetching leaderboard:", error);
      throw new Error("Failed to fetch leaderboard");
    }
  }

  async getUserTag(
    discordID: string
  ): Promise<{ discordID: string; tagNumber: number } | null> {
    try {
      const leaderboard = await this.getLeaderboard();
      const tagEntry = leaderboard.leaderboardData.find(
        (entry) => entry.discordID === discordID
      );
      return tagEntry || null;
    } catch (error) {
      console.error(
        `Error fetching user tag for discordID ${discordID}:`,
        error
      );
      throw new Error(`Could not fetch user tag for discordID ${discordID}`);
    }
  }

  async updateTag(
    discordID: string,
    tagNumber: number
  ): Promise<{ discordID: string; tagNumber: number }> {
    try {
      const currentLeaderboard = await this.getLeaderboard();
      const existingTagIndex = currentLeaderboard.leaderboardData.findIndex(
        (entry) => entry.discordID === discordID
      );

      if (existingTagIndex !== -1) {
        currentLeaderboard.leaderboardData[existingTagIndex].tagNumber =
          tagNumber;
      } else {
        currentLeaderboard.leaderboardData.push({ discordID, tagNumber });
      }

      await this.db
        .update(LeaderboardModel)
        .set({ leaderboardData: currentLeaderboard.leaderboardData })
        .where(
          eq(
            LeaderboardModel.leaderboardID,
            (
              await this.db
                .select({ id: LeaderboardModel.leaderboardID })
                .from(LeaderboardModel)
                .orderBy(asc(LeaderboardModel.leaderboardID))
                .limit(1)
            )[0].id
          )
        )
        .execute();

      return { discordID, tagNumber };
    } catch (error) {
      console.error("Error updating tag:", error);
      throw new Error("Failed to update tag");
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

      await this.db
        .update(LeaderboardModel)
        .set({ leaderboardData })
        .where(
          eq(
            LeaderboardModel.leaderboardID,
            (
              await this.db
                .select({ id: LeaderboardModel.leaderboardID })
                .from(LeaderboardModel)
                .orderBy(asc(LeaderboardModel.leaderboardID))
                .limit(1)
            )[0].id
          )
        )
        .execute();
    } catch (error) {
      console.error("Error processing scores:", error);
      throw new Error("Failed to process scores");
    }
  }
}
