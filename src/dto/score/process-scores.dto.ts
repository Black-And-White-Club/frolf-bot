import {
  IsNotEmpty,
  IsString,
  ValidateNested,
  IsArray,
  IsOptional,
  IsNumber,
} from "class-validator";
import { Type } from "class-transformer";

// Define ScoreInputDto for individual score validation
class ScoreInputDto {
  @IsNotEmpty()
  @IsString()
  discordID!: string;

  @IsNotEmpty()
  @IsNumber() // Use IsNumber for numeric validation
  score!: number;

  @IsOptional() // Use IsOptional if tagNumber is not always required
  @IsNumber({}, { message: "tagNumber must be a number" }) // Validate that it's a number if provided
  tagNumber?: number | null; // Allow it to be null
}

// Define ProcessScoresDto for processing scores
export class ProcessScoresDto {
  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => ScoreInputDto) // Use ScoreInputDto for validation
  scores!: ScoreInputDto[]; // Use ScoreInputDto here
}
