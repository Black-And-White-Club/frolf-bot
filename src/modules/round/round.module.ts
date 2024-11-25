import { Module } from "@nestjs/common";
import { RoundService } from "./round.service";
import { RoundResolver } from "./round.resolver";
@Module({
  providers: [RoundService, RoundResolver],
  exports: [RoundService],
})
export class RoundModule {}
