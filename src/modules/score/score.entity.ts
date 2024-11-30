// src/modules/score/score.entity.ts

import { IsString, IsNumber, IsOptional } from "class-validator";

export class Score {
  @IsString()
  discordID!: string;

  @IsString()
  roundID!: string;

  @IsNumber()
  score!: number;

  @IsNumber()
  @IsOptional()
  tagNumber?: number;

  @IsString()
  createdAt!: Date;

  @IsString()
  updatedAt!: Date;
}
