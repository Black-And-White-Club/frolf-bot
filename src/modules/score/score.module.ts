// src/modules/score/score.module.ts
import { Module } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { ScoreResolver } from "./score.resolver";
import { DatabaseModule } from "../../db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [DatabaseModule.forFeature(schema, "SCORE_DATABASE_CONNECTION")],
  providers: [ScoreResolver, ScoreService],
  exports: [ScoreService],
})
export class ScoreModule {}
