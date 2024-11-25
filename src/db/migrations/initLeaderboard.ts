// src/migrations/initLeaderboard.ts
import { pgTable, integer, varchar } from "drizzle-orm/pg-core";
import { timestamps } from "../helpers/timetamps.helpers";

export const leaderboard = pgTable("leaderboard", {
  discordID: varchar()
    .notNull()
    .unique(),
  tagNumber: integer()
    .notNull()
    .unique(),
  lastPlayed: varchar(),
  durationHeld: integer().default(0),
  ...timestamps,
});
