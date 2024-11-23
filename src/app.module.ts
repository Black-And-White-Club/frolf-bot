import { Module, OnModuleInit, Injectable } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import { YogaDriver } from "@graphql-yoga/nestjs";
import pg from "pg"; // Import pg as default
import { drizzle } from "drizzle-orm/node-postgres"; // Drizzle ORM
import { UserService } from "./services/UserService"; // Import all services from a single path
import { RoundService } from "./services/RoundService";
import { ScoreService } from "./services/ScoreService";
import { LeaderboardService } from "./services/LeaderboardService";
import { UserResolver } from "./resolvers/UserResolver";
import { ScoreResolver } from "./resolvers/ScoreResolver";
import { RoundResolver } from "./resolvers/RoundResolver";
import { LeaderboardResolver } from "./resolvers/LeaderboardResolver";

@Injectable()
export class DatabaseService implements OnModuleInit {
  private pool: pg.Pool; // Use pg.Pool

  constructor() {
    this.pool = new pg.Pool({
      // Create a new Pool instance
      host: process.env.DB_HOST || "localhost", // Use environment variable or default
      port: 5432, // Use environment variable or default
      database: process.env.DB_NAME || "test", // Use environment variable or default
      user: process.env.DB_USER || "postgres", // Use environment variable or default
      password: process.env.DB_PASSWORD || "mypassword", // Use environment variable or default
    });
  }

  async onModuleInit() {
    try {
      await this.pool.query("SELECT 1"); // Simple query to check connection
      console.log("Database connection is successful.");
    } catch (error) {
      console.error("Database connection failed:", error);
    }
  }

  getDrizzleInstance() {
    return drizzle(this.pool); // Return the Drizzle instance
  }
}

@Module({
  imports: [
    GraphQLModule.forRoot({
      driver: YogaDriver,
      typePaths: ["**/*.graphql"],
      context: () => ({}),
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
    ScoreService,
    RoundService,
    LeaderboardService,
    UserResolver,
    ScoreResolver,
    RoundResolver,
    LeaderboardResolver,
  ],
  exports: ["DATABASE_CONNECTION"],
})
export class AppModule {}
