// db/index.ts
import { drizzle } from "drizzle-orm/node-postgres";
import { Pool } from "pg";

// Import User model
import { users } from "../modules/user/user.model";
import { RoundModel } from "../modules/round/round.model";
import { leaderboard } from "./models/Leaderboard";
import { scores } from "./models/Score";
export { users, RoundModel, leaderboard, scores };

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
