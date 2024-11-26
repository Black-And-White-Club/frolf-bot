// src/app.module.ts
import { Module } from "@nestjs/common";
import { UserModule } from "./modules/user/user.module";
import { RoundModule } from "./modules/round/round.module";
import { ScoreModule } from "./modules/score/score.module";
import { LeaderboardModule } from "./modules/leaderboard/leaderboard.module";
import { ApiGatewayModule } from "./api-gateway/api-gateway.module";

@Module({
  imports: [
    ApiGatewayModule,
    UserModule,
    RoundModule,
    ScoreModule,
    LeaderboardModule,
  ],
})
export class AppModule {}
