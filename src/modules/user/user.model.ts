// src/db/models/User.ts
import { pgTable, varchar, serial } from "drizzle-orm/pg-core";
import { timestamps } from "../../db/helpers/timetamps.helpers";
import { UserRole } from "../../enums/user-role.enum";

export const users = pgTable("users", {
  id: serial("id").primaryKey(),
  name: varchar("name"),
  discordID: varchar("discord_id").unique().notNull(),
  role: varchar("role").notNull().default(UserRole.RATTLER),
  ...timestamps,
});
