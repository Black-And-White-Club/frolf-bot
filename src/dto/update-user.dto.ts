// src/dto/update-user.dto.ts

import { IsString, IsInt, IsOptional } from "class-validator";
import { UserRole } from "../enums/user-role.enum"; // Adjust the path according to your structure

export class UpdateUserDto {
  @IsString()
  discordID: string;

  @IsString()
  @IsOptional() // Name is optional in the update
  name?: string;

  @IsInt()
  @IsOptional() // Tag number is optional in the update
  tagNumber?: number;

  @IsString()
  @IsOptional() // Role is optional in the update
  role?: UserRole; // You can also use UserRole enum if you import it

  // Constructor to initialize the required properties
  constructor(discordID: string) {
    this.discordID = discordID;
  }
}
