import {
  IsNotEmpty,
  IsString,
  IsEnum,
  IsOptional,
  IsNumber,
} from "class-validator";
import { Response } from "src/enums";

export class JoinRoundInput {
  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsNotEmpty()
  @IsString()
  discordID!: string;

  @IsNotEmpty()
  @IsEnum(Response, {
    message:
      "Response must be one of the predefined values: ACCEPT, TENTATIVE, DECLINE",
  })
  response!: Response;

  @IsOptional()
  @IsNumber()
  tagNumber?: number; // Make tagNumber optional
}
