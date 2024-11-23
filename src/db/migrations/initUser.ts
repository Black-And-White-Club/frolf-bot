// src/schema.ts
import * as p from "drizzle-orm/pg-core";
import { timestamps } from "../helpers/timetamps.helpers";

export const users = p.pgTable("users", {
  id: p.serial().primaryKey(),
  name: p.text(),
  email: p.text().unique(),
  discordID: p
    .text()
    .unique()
    .notNull(),
  tagNumber: p.integer().unique(),
  role: p.text(),
  ...timestamps, // Include your timestamps if needed
});
