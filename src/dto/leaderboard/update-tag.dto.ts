import { IsNotEmpty, IsString, IsInt, Min } from "class-validator";

export class UpdateTagDto {
  @IsNotEmpty()
  @IsString()
  discordID!: string; // Discord ID of the user

  @IsNotEmpty()
  @IsInt()
  @Min(1) // Assuming tag numbers start from 1
  tagNumber!: number; // The new tag number to assign
}
