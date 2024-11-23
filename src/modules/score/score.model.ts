import { integer, pgTable, varchar } from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";

// Define the Score table
export const scores = pgTable("scores", {
  discordID: varchar().notNull(),
  roundID: varchar().notNull(),
  score: integer().notNull(),
  tagNumber: integer(),
  ...timestamps,
});
