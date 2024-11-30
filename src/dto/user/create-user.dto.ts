import {
  IsString,
  IsOptional,
  IsInt,
  IsNumber,
  IsNotEmpty,
  IsEnum,
} from "class-validator";
import { UserRole } from "src/enums/user-role.enum";

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
  role: UserRole = UserRole.RATTLER;
}
