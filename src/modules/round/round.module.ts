// src/modules/round/round.module.ts
import { Module } from "@nestjs/common";
import { RoundService } from "./round.service";
import { RoundResolver } from "./round.resolver";
import { DatabaseModule } from "../../db/database.module";

@Module({
  imports: [DatabaseModule],
  providers: [RoundResolver, RoundService],
  exports: [RoundService],
})
export class RoundModule {}
