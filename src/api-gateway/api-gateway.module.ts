// src/api-gateway/api-gateway.module.ts
import { Module } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import {
  YogaGatewayDriver,
  YogaGatewayDriverConfig,
} from "@graphql-yoga/nestjs-federation";
import {
  LeaderboardModule,
  RoundModule,
  UserModule,
  ScoreModule,
} from "src/modules";

@Module({
  imports: [
    LeaderboardModule,
    UserModule,
    RoundModule,
    ScoreModule,
    GraphQLModule.forRoot<YogaGatewayDriverConfig>({
      driver: YogaGatewayDriver,
      gateway: {},
      server: {
        cors: true,
        context: ({ req }: { req: Request }) => ({ req }),
      },
    }),
  ],
})
export class ApiGatewayModule {}
