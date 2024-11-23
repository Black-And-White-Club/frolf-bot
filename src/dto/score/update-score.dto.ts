import { IsNotEmpty, IsNumber, IsString, IsOptional } from "class-validator";

export class UpdateScoreDto {
  @IsNotEmpty()
  @IsString()
  discordID!: string;

  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsNotEmpty()
  @IsNumber()
  score!: number;

  @IsOptional() // Mark as optional
  @IsNumber({}, { message: "tagNumber must be a number" }) // Validate that it's a number if provided
  tagNumber?: number | null; // Allow it to be null
}
