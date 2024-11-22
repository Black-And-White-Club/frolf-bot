// db/index.ts
import { pgTable, serial, text, varchar } from "drizzle-orm/pg-core";
import { drizzle } from "drizzle-orm/node-postgres";
import { Pool } from "pg";

// Import User model
import { User } from "./models/User";

export { User };

// Initialize the database connection
const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});
export const db = drizzle({ client: pool });

// Provide utility functions for specific queries
export async function testDbConnection() {
  return db.execute("select 1");
}
