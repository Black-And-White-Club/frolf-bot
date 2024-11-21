// src/dto/create-user.dto.ts
import {
  IsString,
  IsInt,
  IsOptional,
  IsEnum,
  IsNotEmpty,
} from "class-validator";
import { UserRole } from "../enums/user-role.enum"; // Adjust the path according to your structure

export class CreateUserDto {
  @IsString()
  @IsNotEmpty()
  name!: string;

  @IsString()
  @IsNotEmpty()
  discordID!: string;

  @IsInt()
  @IsOptional()
  tagNumber?: number;

  @IsEnum(UserRole)
  @IsNotEmpty()
  role!: UserRole;
}
