// src/db/models/user.model.ts
import { pgTable, varchar, serial, integer } from "drizzle-orm/pg-core";
import { timestamps } from "src/db/helpers/timetamps.helpers";
import { UserRole } from "src/enums";

export const users = pgTable("users", {
  id: serial("id").primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  discordID: varchar("discord_id", { length: 256 }).unique().notNull(),
  role: varchar("role", { length: 32 }).notNull().default(UserRole.Rattler),
  tagNumber: integer("tag_number"),
  ...timestamps,
});
