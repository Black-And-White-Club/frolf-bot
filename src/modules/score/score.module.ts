import { Module } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { ScoreController } from "./score.controller";
import { DatabaseModule } from "src/db/database.module";
import * as schema from "src/schema";
import { ConsumerService } from "src/rabbitmq/consumer";

@Module({
  imports: [DatabaseModule.forFeature(schema, "SCORE_DATABASE_CONNECTION")],
  controllers: [ScoreController],
  providers: [ScoreService, ConsumerService],
  exports: [ScoreService],
})
export class ScoreModule {}
