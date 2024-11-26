// src/db/database.module.ts

import { Module, DynamicModule, Provider } from "@nestjs/common";
import { drizzle } from "drizzle-orm/node-postgres";
import { Pool } from "pg";
import * as schema from "src/schema"; // Import the combined schema

const connectionString =
  process.env.DATABASE_URL ||
  "postgres://postgres:mypassword@localhost:5432/test";
const pool = new Pool({ connectionString });

@Module({})
export class DatabaseModule {
  static forFeature(
    schema: any, // Pass the specific schema for the module
    providerName: string // Provide a unique provider name
  ): DynamicModule {
    const dbProvider: Provider = {
      provide: providerName,
      useValue: drizzle(pool, { schema }),
    };

    return {
      module: DatabaseModule,
      providers: [dbProvider],
      exports: [dbProvider],
    };
  }
}
