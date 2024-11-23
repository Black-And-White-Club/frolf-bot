import { IsString, IsInt, IsNotEmpty } from "class-validator";

export class LinkTagDto {
  @IsString()
  @IsNotEmpty()
  discordID!: string;

  @IsInt()
  newTagNumber!: number;
}
