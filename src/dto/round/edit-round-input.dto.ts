import { IsString, IsOptional, IsDateString } from "class-validator";

export class EditRoundInput {
  @IsOptional()
  @IsString()
  title?: string;

  @IsOptional()
  @IsString()
  location?: string;

  @IsOptional()
  @IsString()
  eventType?: string;

  @IsOptional()
  @IsDateString({}, { message: "Date must be in valid format (YYYY-MM-DD)" })
  date?: string;

  @IsOptional()
  @IsString()
  time?: string;
}
