// src/dto/round/round-input.dto.ts
import {
  IsNotEmpty,
  IsString,
  IsOptional,
  IsDateString,
} from "class-validator";

export class ScheduleRoundInput {
  @IsNotEmpty()
  @IsString()
  title!: string;

  @IsNotEmpty()
  @IsString()
  location!: string;

  @IsOptional()
  @IsString()
  eventType: string = "Round";

  @IsNotEmpty()
  @IsDateString({}, { message: "Date must be in valid format (YYYY-MM-DD)" })
  date!: string;

  @IsNotEmpty()
  @IsString()
  time!: string;

  @IsNotEmpty()
  @IsString()
  creatorID!: string;
}
