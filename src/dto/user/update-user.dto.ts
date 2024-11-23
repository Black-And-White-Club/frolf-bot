import {
  IsString,
  IsInt,
  IsOptional,
  IsEnum,
  IsNotEmpty,
} from "class-validator";
import { UserRole } from "../../enums/user-role.enum";

export class UpdateUserDto {
  @IsOptional() // Make name optional for updates
  @IsString()
  name?: string;

  @IsNotEmpty({ message: "DiscordID should not be empty" })
  @IsString()
  discordID!: string; // This field should always be required for updates

  @IsOptional() // Make tagNumber optional for updates
  @IsInt()
  tagNumber?: number | null; // Allow tagNumber to be a number or null

  @IsEnum(UserRole)
  role!: UserRole; // Ensure that the role is always provided
}
