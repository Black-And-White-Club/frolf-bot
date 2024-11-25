import { Module, OnModuleInit, Injectable } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import { YogaDriver } from "@graphql-yoga/nestjs";
import pg from "pg"; // Import pg as default
import { drizzle } from "drizzle-orm/node-postgres"; // Drizzle ORM
import { UserService } from "./services/UserService"; // Import services
import { UserResolver } from "./resolvers/UserResolver"; // Import resolvers
import { join } from "path"; // Import path module

@Injectable()
export class DatabaseService implements OnModuleInit {
  private pool: pg.Pool;

  constructor() {
    this.pool = new pg.Pool({
      host: process.env.DB_HOST || "localhost",
      port: 5432,
      database: process.env.DB_NAME || "test",
      user: process.env.DB_USER || "postgres",
      password: process.env.DB_PASSWORD || "mypassword",
    });
  }

  async onModuleInit() {
    try {
      await this.pool.query("SELECT 1");
      console.log("Database connection is successful.");
    } catch (error) {
      console.error("Database connection failed:", error);
    }
  }

  getDrizzleInstance() {
    return drizzle(this.pool);
  }
}

@Module({
  imports: [
    GraphQLModule.forRoot({
      driver: YogaDriver,
      typePaths: [join(__dirname, "**/*.graphql")], // Adjust path to your schema
      context: ({ req }) => ({
        // You can pass any context you need here
      }),
    }),
  ],
  providers: [
    {
      provide: "DATABASE_CONNECTION",
      useFactory: (databaseService: DatabaseService) =>
        databaseService.getDrizzleInstance(),
      inject: [DatabaseService],
    },
    UserService,
    UserResolver,
  ],
  exports: ["DATABASE_CONNECTION"],
})
export class AppModule {}
