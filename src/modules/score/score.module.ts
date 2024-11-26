// src/modules/score/score.module.ts
import { Module } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { ScoreResolver } from "./score.resolver";
import { DatabaseModule } from "../../db/database.module";

@Module({
  imports: [DatabaseModule],
  providers: [ScoreResolver, ScoreService],
  exports: [ScoreService],
})
export class ScoreModule {}
