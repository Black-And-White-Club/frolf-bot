import {
  pgTable,
  serial,
  varchar,
  json,
  boolean,
  date,
  time,
} from "drizzle-orm/pg-core";
import { timestamps } from "../helpers/timetamps.helpers";

export const rounds = pgTable("rounds", {
  roundID: serial().primaryKey(),
  title: varchar().notNull(),
  location: varchar().notNull(),
  eventType: varchar(),
  date: date().notNull(),
  time: time().notNull(),
  participants: json()
    .notNull()
    .default(JSON.stringify([])), // Default to empty array
  scores: json()
    .notNull()
    .default(JSON.stringify([])), // Default to empty array
  finalized: boolean().default(false),
  creatorID: varchar().notNull(),
  state: varchar().notNull(),
  ...timestamps,
});
