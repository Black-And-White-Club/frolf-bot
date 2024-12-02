// src/modules/leaderboard/leaderboard.module.ts

import { Module } from "@nestjs/common";
import { LeaderboardService } from "./leaderboard.service";
import { LeaderboardController } from "./leaderboard.controller";
import { DatabaseModule } from "src/db/database.module";
import { QueueService } from "src/rabbitmq/queue.service";
import { Publisher } from "src/rabbitmq/publisher";
import { MessagingModule } from "src/rabbitmq/messaging.module";
import { RabbitMQModule } from "@golevelup/nestjs-rabbitmq";

@Module({
  imports: [
    DatabaseModule.forFeature({}, "LEADERBOARD_DATABASE_CONNECTION"),
    MessagingModule,
    RabbitMQModule.forRoot(RabbitMQModule, {
      exchanges: [{ name: "main_exchange", type: "direct" }],
      uri: "amqp://localhost:5672",
      connectionInitOptions: { wait: false },
    }),
  ],
  controllers: [LeaderboardController],
  providers: [LeaderboardService, QueueService, Publisher],
  exports: [LeaderboardService],
})
export class LeaderboardModule {}
