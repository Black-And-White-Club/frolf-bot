import { IsNotEmpty, IsString, IsEnum } from "class-validator";
import { Response } from "../../enums/round-enum"; // Assuming you have this enum defined

export class JoinRoundInput {
  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsNotEmpty()
  @IsString()
  discordID!: string; // Include discordID

  @IsNotEmpty()
  @IsEnum(Response, {
    message:
      "Response must be one of the predefined values: ACCEPT, TENTATIVE, DECLINE",
  })
  response!: Response; // Use the Response enum
}
