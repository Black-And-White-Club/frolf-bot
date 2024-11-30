// src/modules/score/score.module.ts

import { Module } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { ScoreController } from "./score.controller";
import { DatabaseModule } from "src/db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [DatabaseModule.forFeature(schema, "SCORE_DATABASE_CONNECTION")],
  controllers: [ScoreController],
  providers: [ScoreService],
  exports: [ScoreService],
})
export class ScoreModule {}
