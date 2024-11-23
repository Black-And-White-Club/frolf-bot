import { Injectable, Inject } from "@nestjs/common";
import { leaderboard as LeaderboardModel } from "../db/models/Leaderboard";
import { eq, asc } from "drizzle-orm"; // Changed from 'desc' to 'asc' for golf scores
import { TagNumber } from "../types.generated";
import { drizzle } from "drizzle-orm/node-postgres";

@Injectable()
export class LeaderboardService {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>
  ) {
    console.log("Injected DB instance:", db);
  }

  async getLeaderboard(
    page: number,
    limit: number
  ): Promise<{ users: TagNumber[] }> {
    const offset = (page - 1) * limit;

    try {
      const leaderboardEntries = await this.db
        .select()
        .from(LeaderboardModel)
        .orderBy(asc(LeaderboardModel.tagNumber)) // Use ascending order for golf scores
        .limit(limit)
        .offset(offset)
        .execute();

      const users: TagNumber[] = leaderboardEntries.map((entry) => ({
        __typename: "TagNumber" as const,
        discordID: entry.discordID,
        tagNumber: entry.tagNumber,
        lastPlayed: entry.lastPlayed || "",
        durationHeld: entry.durationHeld || 0,
      }));

      return { users };
    } catch (error) {
      console.error("Error fetching leaderboard:", error);
      throw new Error("Could not fetch leaderboard");
    }
  }

  async linkTag(discordID: string, newTagNumber: number): Promise<TagNumber> {
    if (!discordID) {
      throw new Error("discordID cannot be empty.");
    }

    // Check if the tag number is already taken
    const existingTag = await this.getUserByTagNumber(newTagNumber);
    if (existingTag) {
      throw new Error(`Tag number ${newTagNumber} is already taken.`);
    }

    // Check if the user already has a tag
    const existingUserTag = await this.getUserTag(discordID);
    if (existingUserTag) {
      // Update the existing tag
      await this.db
        .update(LeaderboardModel)
        .set({ tagNumber: newTagNumber, lastPlayed: new Date().toISOString() })
        .where(eq(LeaderboardModel.discordID, discordID))
        .execute();
    } else {
      // Insert a new tag for the user
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

    // Return the linked tag without fetching it again
    return {
      __typename: "TagNumber",
      discordID,
      tagNumber: newTagNumber,
      lastPlayed: new Date().toISOString(),
      durationHeld: 0,
    };
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
          __typename: "TagNumber",
          discordID: entry.discordID,
          tagNumber: entry.tagNumber,
          lastPlayed: entry.lastPlayed || "",
          durationHeld: entry.durationHeld ?? 0, // Default to 0 if null
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
          __typename: "TagNumber",
          discordID: entry.discordID,
          tagNumber: entry.tagNumber,
          lastPlayed: entry.lastPlayed || "",
          durationHeld: entry.durationHeld ?? 0, // Default to 0 if null
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
        __typename: "TagNumber",
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
    const processedTags: TagNumber[] = [];

    for (const scoreInput of scores) {
      try {
        const existingTag = await this.getUserTag(scoreInput.discordID);

        if (existingTag) {
          // Only update the tag if the new score is better (lower) than the existing tag number
          if (scoreInput.score < existingTag.tagNumber) {
            await this.updateTag(
              scoreInput.discordID,
              scoreInput.tagNumber || scoreInput.score // Use the new score as the tag number
            );

            processedTags.push({
              __typename: "TagNumber",
              discordID: scoreInput.discordID,
              tagNumber: scoreInput.tagNumber || scoreInput.score,
              lastPlayed: new Date().toISOString(), // Update last played
              durationHeld: existingTag.durationHeld, // Keep the existing duration held
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
      } catch (error) {
        console.error(
          `Error processing score for ${scoreInput.discordID}:`,
          error
        );
        throw new Error(`Could not process score for ${scoreInput.discordID}`);
      }
    }

    return processedTags;
  }
}
