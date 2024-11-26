// src/modules/leaderboard/leaderboard.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { leaderboard as LeaderboardModel } from "../../schema";
import { eq, asc } from "drizzle-orm";
import { TagNumber } from "../../types.generated";
import { NodePgDatabase } from "drizzle-orm/node-postgres";

@Injectable()
export class LeaderboardService {
  constructor(@Inject("DATABASE_CONNECTION") private db: NodePgDatabase) {}

  async getLeaderboard(
    page: number,
    limit: number
  ): Promise<{ users: TagNumber[] }> {
    try {
      console.log("Service received page:", page, "limit:", limit);
      if (typeof page !== "number" || typeof limit !== "number") {
        throw new Error("Page and limit must be numbers");
      }
      const offset = (page - 1) * limit;

      const leaderboardEntries = await this.db
        .select()
        .from(LeaderboardModel)
        .orderBy(asc(LeaderboardModel.tagNumber))
        .limit(limit)
        .offset(offset);

      const users: TagNumber[] = leaderboardEntries.map((entry) => ({
        ...entry,
        lastPlayed: entry.lastPlayed ? entry.lastPlayed.toString() : "",
        durationHeld: entry.durationHeld ?? 0,
      }));

      return { users };
    } catch (error) {
      console.error("Error fetching leaderboard:", error);
      throw new Error("Failed to fetch leaderboard");
    }
  }

  async linkTag(discordID: string, newTagNumber: number): Promise<TagNumber> {
    try {
      if (!discordID) {
        throw new Error("discordID cannot be empty.");
      }

      const existingTag = await this.getUserByTagNumber(newTagNumber);
      if (existingTag) {
        throw new Error(`Tag number ${newTagNumber} is already taken.`);
      }

      const existingUserTag = await this.getUserTag(discordID);
      if (existingUserTag) {
        await this.db
          .update(LeaderboardModel)
          .set({
            tagNumber: newTagNumber,
            lastPlayed: new Date().toISOString(),
          })
          .where(eq(LeaderboardModel.discordID, discordID))
          .execute();
      } else {
        await this.db
          .insert(LeaderboardModel)
          .values({
            discordID,
            tagNumber: newTagNumber,
            lastPlayed: new Date().toISOString(),
            durationHeld: 0,
          })
          .execute();
      }

      return {
        discordID,
        tagNumber: newTagNumber,
        lastPlayed: new Date().toISOString(),
        durationHeld: 0,
      };
    } catch (error) {
      console.error("Error linking tag:", error);
      throw new Error("Failed to link tag");
    }
  }

  async getUserTag(discordID: string): Promise<TagNumber | null> {
    try {
      console.log(`Fetching tag for discordID: ${discordID}`);

      const tagEntry = await this.db
        .select()
        .from(LeaderboardModel)
        .where(eq(LeaderboardModel.discordID, discordID))
        .execute();

      if (tagEntry.length > 0) {
        const entry = tagEntry[0];
        return {
          discordID: entry.discordID,
          tagNumber: entry.tagNumber,
          lastPlayed: entry.lastPlayed ? entry.lastPlayed.toString() : "",
          durationHeld: entry.durationHeld ?? 0,
        };
      }

      console.warn(`No tag found for discordID: ${discordID}`);
      return null;
    } catch (error) {
      console.error(
        `Error fetching user tag for discordID ${discordID}:`,
        error
      );
      throw new Error(`Could not fetch user tag for discordID ${discordID}`);
    }
  }

  async getUserByTagNumber(tagNumber: number): Promise<TagNumber | null> {
    try {
      const tagEntry = await this.db
        .select()
        .from(LeaderboardModel)
        .where(eq(LeaderboardModel.tagNumber, tagNumber))
        .execute();

      if (tagEntry.length > 0) {
        const entry = tagEntry[0];
        return {
          discordID: entry.discordID,
          tagNumber: entry.tagNumber,
          lastPlayed: entry.lastPlayed ? entry.lastPlayed.toString() : "",
          durationHeld: entry.durationHeld ?? 0,
        };
      }

      return null;
    } catch (error) {
      console.error(`Error fetching user by tagNumber ${tagNumber}:`, error);
      throw new Error(`Could not fetch user by tagNumber ${tagNumber}`);
    }
  }

  async updateTag(discordID: string, tagNumber: number): Promise<TagNumber> {
    try {
      const existingTag = await this.getUserTag(discordID);
      if (existingTag) {
        await this.db
          .update(LeaderboardModel)
          .set({ tagNumber })
          .where(eq(LeaderboardModel.discordID, discordID))
          .execute();
      } else {
        await this.db
          .insert(LeaderboardModel)
          .values({
            discordID,
            tagNumber,
            lastPlayed: new Date().toISOString(),
            durationHeld: 0,
          })
          .execute();
      }

      return {
        discordID,
        tagNumber,
        lastPlayed: new Date().toISOString(),
        durationHeld: 0,
      };
    } catch (error) {
      if (error instanceof Error) {
        console.error("Error updating tag:", error);
        throw new Error(
          `Could not update tag for discordID ${discordID}: ${error.message}`
        );
      } else {
        console.error("Unknown error updating tag:", error);
        throw new Error(
          `Could not update tag for discordID ${discordID}: Unknown error`
        );
      }
    }
  }

  async processScores(
    scores: { discordID: string; score: number; tagNumber?: number | null }[]
  ): Promise<TagNumber[]> {
    try {
      const processedTags: TagNumber[] = [];

      for (const scoreInput of scores) {
        const existingTag = await this.getUserTag(scoreInput.discordID);

        if (existingTag) {
          if (scoreInput.score < existingTag.tagNumber) {
            await this.updateTag(
              scoreInput.discordID,
              scoreInput.tagNumber || scoreInput.score
            );

            processedTags.push({
              discordID: scoreInput.discordID,
              tagNumber: scoreInput.tagNumber || scoreInput.score,
              lastPlayed: new Date().toISOString(),
              durationHeld: existingTag.durationHeld,
            });
          } else {
            console.warn(
              `New score (${scoreInput.score}) is not better than existing tag (${existingTag.tagNumber}) for discordID: ${scoreInput.discordID}. Skipping update.`
            );
          }
        } else {
          console.warn(
            `No tag found for discordID: ${scoreInput.discordID}. Skipping update.`
          );
        }
      }

      return processedTags;
    } catch (error) {
      console.error("Error processing scores:", error);
      throw new Error("Failed to process scores");
    }
  }
}
