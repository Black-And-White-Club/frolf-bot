import { IsNotEmpty, IsNumber, IsString } from "class-validator";

export class SubmitScoreDto {
  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsNotEmpty()
  @IsNumber()
  score!: number;

  @IsNotEmpty()
  @IsNumber()
  tagNumber!: number; // Assuming tagNumber is required for score submission
}
