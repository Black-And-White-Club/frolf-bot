import {
  pgTable,
  serial,
  varchar,
  json,
  boolean,
  date,
  time,
} from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";

export const RoundModel = pgTable("rounds", {
  roundID: serial("round_id").primaryKey(), // Use snake_case for column names
  title: varchar("title").notNull(),
  location: varchar("location").notNull(),
  eventType: varchar("event_type"),
  date: date("date").notNull(),
  time: time("time").notNull(),
  participants: json("participants").notNull().default(JSON.stringify([])),
  scores: json("scores").notNull().default(JSON.stringify([])),
  finalized: boolean("finalized").default(false),
  creatorID: varchar("creator_id").notNull(), // Use snake_case for column names
  state: varchar("state").notNull(),
  ...timestamps,
});
