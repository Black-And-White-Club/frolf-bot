// src/db/database.ts
import pg from "pg"; // Import the entire pg namespace
import { drizzle } from "drizzle-orm/node-postgres";
import * as schema from "./schema";

const connectionString =
  process.env.DATABASE_URL ||
  "postgres://postgres:mypassword@localhost:5432/test";

// Access the Pool class from the pg namespace
const { Pool } = pg;
const pool = new Pool({ connectionString });

export const db = drizzle(pool, { schema });
