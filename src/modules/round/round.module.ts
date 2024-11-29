// src/modules/round/round.module.ts
import { Module } from "@nestjs/common";
import { RoundService } from "./round.service";
import { RoundResolver } from "./round.resolver";
import { DatabaseModule } from "../../db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [DatabaseModule.forFeature(schema, "ROUND_DATABASE_CONNECTION")],
  providers: [RoundResolver, RoundService],
  exports: [RoundService],
})
export class RoundModule {}
