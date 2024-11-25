import {
  IsString,
  IsOptional,
  IsInt,
  IsEnum,
  IsNotEmpty,
  IsDate,
  IsEmpty,
} from "class-validator";
import { UserRole } from "../../enums/user-role.enum";

export class CreateUserDto {
  @IsString()
  @IsNotEmpty()
  discordID!: string;

  @IsString()
  name!: string;

  @IsInt()
  @IsOptional()
  tagNumber: number | null = null; // Adjust this to allow null

  @IsEnum(UserRole)
  role!: UserRole;

  @IsDate()
  createdAt!: Date;

  @IsDate()
  updatedAt?: Date;
}
