import {
  IsString,
  IsOptional,
  IsInt,
  IsNumber,
  IsNotEmpty,
  IsEnum,
} from "class-validator";
import { UserRole } from "src/enums";

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
  @IsNotEmpty() // Ensure the role is provided
  role!: UserRole; // Make role non-optional, as it seems required
}
