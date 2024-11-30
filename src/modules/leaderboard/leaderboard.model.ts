// src/db/models/leaderboard.model.ts
import { pgTable, serial, jsonb, boolean } from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";
import { LeaderboardEntry } from "src/modules/leaderboard/leaderboard-entry.entity";

export const leaderboard = pgTable("leaderboard", {
  leaderboardID: serial("leaderboard_id").primaryKey(),
  leaderboardData: jsonb("leaderboard_data").$type<LeaderboardEntry[]>(),
  active: boolean("active").notNull().default(true), // Add 'active' column
  ...timestamps,
});
