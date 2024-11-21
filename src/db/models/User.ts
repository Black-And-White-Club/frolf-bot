// src/db/models/User.ts
import { pgTable, serial, varchar, integer } from "drizzle-orm/pg-core";

export const User = pgTable("users", {
  name: varchar("name"),
  discordID: varchar("discordID").unique().notNull(),
  tagNumber: integer("tagNumber"),
  role: varchar("role"),
});
