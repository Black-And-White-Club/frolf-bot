// src/app.module.ts

import { Module } from "@nestjs/common";
import { RabbitMQModule } from "@golevelup/nestjs-rabbitmq";
import * as modules from "./modules";

@Module({
  imports: [
    modules.UserModule,
    modules.RoundModule,
    modules.ScoreModule,
    modules.LeaderboardModule,
    RabbitMQModule.forRoot(RabbitMQModule, {
      exchanges: [
        // ... your exchange configuration
      ],
      uri: "amqp://localhost:5672",
      connectionInitOptions: { wait: false },
    }),
  ],
})
export class AppModule {}
