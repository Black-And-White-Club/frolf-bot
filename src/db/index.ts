// db/index.ts
import { drizzle } from "drizzle-orm/node-postgres";
import { Pool } from "pg";

// Import User model
import { User } from "./models/User";

export { User };

// Function to create a new database connection
export function createDbClient(connectionDetails: {
  host: string;
  port: number;
  database: string;
  user: string;
  password: string;
}) {
  const pool = new Pool(connectionDetails);
  return drizzle({ client: pool });
}

// Provide utility functions for specific queries
export async function testDbConnection(db: ReturnType<typeof drizzle>) {
  return db.execute("select 1");
}
