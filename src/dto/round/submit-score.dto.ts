// src/dto/round/submit-score.dto.ts
import { IsNotEmpty, IsNumber, IsString, IsOptional } from "class-validator";

export class SubmitScoreDto {
  @IsNotEmpty()
  @IsString()
  discordID!: string;

  @IsNotEmpty()
  @IsNumber()
  score!: number;

  @IsOptional()
  @IsNumber()
  tagNumber?: number;
}
