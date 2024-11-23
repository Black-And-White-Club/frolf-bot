import {
  IsArray,
  IsNotEmpty,
  ValidateNested,
  IsString,
  IsInt,
  IsOptional,
} from "class-validator";
import { Type } from "class-transformer";

class ScoreData {
  @IsNotEmpty()
  @IsInt()
  score!: number; // The score achieved by the user

  @IsNotEmpty()
  @IsString()
  discordID!: string; // The Discord ID associated with the score

  @IsOptional() // Tag number is optional
  @IsInt()
  tagNumber?: number; // The current tag number associated with the user
}

export class ReceiveScoresDto {
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => ScoreData) // Use class-transformer to validate nested objects
  scores!: ScoreData[]; // Array of score data to be processed
}
