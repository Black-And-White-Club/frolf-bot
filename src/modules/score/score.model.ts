import { integer, pgTable, varchar } from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";

export const scores = pgTable("scores", {
  discordID: varchar("discord_id").notNull(),
  roundID: varchar("round_id").notNull(),
  score: integer("score").notNull(),
  tagNumber: integer("tag_number"),
  ...timestamps,
});
