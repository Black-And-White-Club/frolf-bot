// src/app.module.ts
import { Module } from "@nestjs/common";
import { RouterModule } from "@nestjs/core";
import { UserModule } from "./modules/user/user.module";
import { RoundModule } from "./modules/round/round.module";
import { ScoreModule } from "./modules/score/score.module";
import { LeaderboardModule } from "./modules/leaderboard/leaderboard.module";
import { ApiGatewayModule } from "./modules/api-gateway/api-gateway.module";

@Module({
  imports: [
    RouterModule.register([
      {
        path: "/v1/user",
        module: UserModule,
      },
      {
        path: "/v1/gateway",
        module: ApiGatewayModule,
      },
    ]),
    ApiGatewayModule,
    UserModule,
    RoundModule,
    ScoreModule,
    LeaderboardModule,
  ],
})
export class AppModule {}
