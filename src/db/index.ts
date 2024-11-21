// Make sure to install the 'pg' package
import { pgTable, serial, text, varchar } from "drizzle-orm/pg-core";
import { drizzle } from "drizzle-orm/node-postgres";
import { Pool } from "pg";
import { User } from "./models/User"; // Adjust the import path as needed

export { User };

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});
export const db = drizzle({ client: pool });

export const result = await db.execute("select 1");
