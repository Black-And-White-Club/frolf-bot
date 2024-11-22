// src/db/models/Score.ts
import { pgTable, varchar, integer, uniqueIndex } from "drizzle-orm/pg-core";

export const Score = pgTable("scores", {
  discordID: varchar("discordID").notNull(), // Ensure discordID is not null
  roundID: varchar("roundID").notNull(), // Ensure roundID is not null
  score: integer("score").notNull(), // Ensure score is not null
  tagNumber: integer("tagNumber"), // Optional tag field
});

// Define a unique index for the combination of discordID and roundID
export const ScoreUniqueIndex = uniqueIndex("unique_round").on(
  Score.discordID,
  Score.roundID
);
