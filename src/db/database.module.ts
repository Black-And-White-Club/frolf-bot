// src/db/database.module.ts

import { Module, DynamicModule, Provider } from "@nestjs/common";
import { drizzle } from "drizzle-orm/node-postgres";
import pg from "pg";

const connectionString =
  process.env.DATABASE_URL ||
  "postgres://postgres:mypassword@localhost:5432/test";

@Module({})
export class DatabaseModule {
  static forFeature(schema: any, providerName: string): DynamicModule {
    const dbProvider: Provider = {
      provide: providerName,
      useFactory: () => {
        // Use a factory function
        const pool = new pg.Pool({ connectionString });
        return drizzle(pool, { schema });
      },
      inject: [],
    };

    return {
      module: DatabaseModule,
      providers: [dbProvider],
      exports: [dbProvider],
    };
  }
}
