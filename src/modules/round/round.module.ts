// src/modules/round/round.module.ts

import { Module } from "@nestjs/common";
import { RoundService } from "./round.service";
import { RoundController } from "./round.controller";
import { DatabaseModule } from "src/db/database.module";
import * as schema from "src/schema";

@Module({
  imports: [DatabaseModule.forFeature(schema, "ROUND_DATABASE_CONNECTION")],
  controllers: [RoundController],
  providers: [RoundService],
  exports: [RoundService],
})
export class RoundModule {}
