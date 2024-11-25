// db/index.ts
import { drizzle } from "drizzle-orm/node-postgres";
import pg from "pg";

// Import User model
import { users } from "../modules/user/user.model";
import { RoundModel } from "../modules/round/round.model";
import { leaderboard } from "../modules/leaderboard/leaderboard.model";
import { scores } from "../modules/score/score.model";
export { users, RoundModel, leaderboard, scores };
const { Pool } = pg;

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

export const databaseProvider = {
  provide: "DATABASE", // Choose a unique provider name
  useFactory: () =>
    createDbClient({
      host: process.env.DB_HOST || "localhost",
      port: Number(process.env.DB_PORT) || 5432,
      database: process.env.DB_NAME || "test",
      user: process.env.DB_USER || "postgres",
      password: process.env.DB_PASSWORD || "mypassword",
    }),
};
