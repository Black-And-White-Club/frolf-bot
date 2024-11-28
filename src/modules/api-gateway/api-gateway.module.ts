// src/modules/api-gateway/api-gateway.module.ts
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
import { ApiGatewayService } from "./api-gateway.service";
import { NestFactory } from "@nestjs/core";

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
        context: async ({ req }) => {
          const discordID = req.headers["discord-id"] as string;
          if (discordID) {
            const apiGatewayService = await req.get("apiGatewayService");
            const user = await apiGatewayService.getUserByDiscordID(discordID);
            return { req, user };
          }
          return { req };
        },
      },
    }),
  ],
  providers: [ApiGatewayService],
})
export class ApiGatewayModule {
  constructor(private readonly apiGatewayService: ApiGatewayService) {}

  async onModuleInit() {
    const app = await NestFactory.create(ApiGatewayModule);

    const yogaServer = app.get(GraphQLModule).getGraphQlServer();

    // Use a plugin to inject the service into the request context
    yogaServer.addPlugin({
      async requestDidStart() {
        return {
          async willSendResponse(responseContext) {
            const req = responseContext.request.request as any;
            req.set("apiGatewayService", this.apiGatewayService);
          },
        };
      },
    });

    await app.init();
  }
}
