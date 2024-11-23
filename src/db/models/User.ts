// src/db/models/User.ts
import { pgTable, varchar, integer } from "drizzle-orm/pg-core";
import { timestamps } from "./timetamps.helpers";

export const users = pgTable("users", {
  name: varchar(),
  discordID: varchar()
    .unique()
    .notNull(),
  tagNumber: integer().unique(),
  role: varchar(),
  ...timestamps,
});
