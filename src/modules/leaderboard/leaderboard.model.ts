import { pgTable, serial, jsonb } from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";

export const leaderboard = pgTable("leaderboard", {
  leaderboardID: serial().primaryKey(),
  leaderboardData:
    jsonb().$type<Array<{ discordID: string; tagNumber: number }>>(),
  ...timestamps,
});
