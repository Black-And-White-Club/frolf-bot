import { Module } from "@nestjs/common";
import { ScoreService } from "./score.service";
import { ScoreResolver } from "./score.resolver";

@Module({
  providers: [ScoreService, ScoreResolver],
  exports: [ScoreService],
})
export class ScoreModule {}
