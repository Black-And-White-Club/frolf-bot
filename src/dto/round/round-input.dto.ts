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
  eventType?: string;

  @IsNotEmpty()
  @IsDateString({}, { message: "Date must be in valid format (YYYY-MM-DD)" })
  date!: string;

  @IsNotEmpty()
  @IsString()
  time!: string; // Ensure proper time format if needed, i.e. "HH:mm"

  @IsNotEmpty()
  @IsString()
  creatorID!: string;
}
