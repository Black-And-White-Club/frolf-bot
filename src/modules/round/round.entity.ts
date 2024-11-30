// src/modules/round/round.entity.ts
import {
  IsString,
  IsNumber,
  IsOptional,
  IsEnum,
  IsDate,
  IsArray,
} from "class-validator";
import { RoundState, Response } from "src/enums";

export class Participant {
  @IsString()
  discordID!: string;

  @IsEnum(Response)
  response!: Response;

  @IsNumber()
  @IsOptional()
  tagNumber?: number | null;
}

export class Round {
  @IsNumber()
  roundID!: number;

  @IsString()
  title!: string;

  @IsString()
  location!: string;

  @IsString()
  @IsOptional()
  eventType?: string;

  @IsDate()
  date!: Date;

  @IsString() // Assuming time is stored as a string
  time!: string;

  @IsArray()
  participants!: Participant[];

  @IsArray()
  scores!: any[]; // Define the structure of your scores

  @IsOptional()
  finalized: boolean = false;

  @IsString()
  creatorID!: string;

  @IsEnum(RoundState)
  state: RoundState = RoundState.Upcoming;

  @IsString()
  createdAt!: string;

  @IsString()
  updatedAt!: string;
}
