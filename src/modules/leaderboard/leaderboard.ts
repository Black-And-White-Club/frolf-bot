import { integer, pgTable, varchar } from "drizzle-orm/pg-core";
import { timestamps } from "../helpers/timetamps.helpers";

// Define the Leaderboard table
export const leaderboard = pgTable("leaderboard", {
  discordID: varchar()
    .notNull()
    .unique(), // Each discordID must be unique
  tagNumber: integer()
    .notNull()
    .unique(), // Each tagNumber must be unique
  lastPlayed: varchar(),
  durationHeld: integer().default(0),
  ...timestamps, // Spread the timestamps into the model
});
