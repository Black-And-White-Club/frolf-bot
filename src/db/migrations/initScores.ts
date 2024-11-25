// src/migrations/initScores.ts
import { integer, pgTable, varchar } from "drizzle-orm/pg-core";
import { timestamps } from "../helpers/timetamps.helpers";

export const scores = pgTable("scores", {
  discordID: varchar().notNull(),
  roundID: varchar().notNull(),
  score: integer().notNull(),
  tagNumber: integer(),
  ...timestamps,
});
