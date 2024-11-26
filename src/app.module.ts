// src/app.module.ts
import { Module } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import { YogaDriver } from "@graphql-yoga/nestjs";
import { GraphQLError, GraphQLFormattedError } from "graphql";
import { UserModule } from "./modules/user/user.module";
import { RoundModule } from "./modules/round/round.module";
import { ScoreModule } from "./modules/score/score.module";
import { LeaderboardModule } from "./modules/leaderboard/leaderboard.module";
import { readFileSync } from "fs";
import { join } from "path";

// Get the directory of the current file using import.meta.url
const __dirname = new URL(".", import.meta.url).pathname;

// Load the schema from schema.generated.graphqls
const typeDefs = readFileSync(join(__dirname, "./schema.generated.graphqls"));

// Import createResolvers
import { createResolvers } from "./modules";
import { db } from "./database";

@Module({
  imports: [
    GraphQLModule.forRoot({
      driver: YogaDriver,
      typeDefs,
      resolvers: createResolvers(db as any),
      context: (ctx: any) => ({
        ...ctx,
      }),
      formatError: (error: GraphQLError) => {
        const graphQLFormattedError: GraphQLFormattedError = {
          message: error.message,
        };
        return graphQLFormattedError;
      },
    }),
    UserModule,
    RoundModule,
    ScoreModule,
    LeaderboardModule,
  ],
})
export class AppModule {}
