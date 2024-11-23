import { IsNotEmpty, IsString } from "class-validator";
import { Response } from "../../types.generated"; // Assuming you have this enum defined

export class JoinRoundInput {
  @IsNotEmpty()
  @IsString()
  roundID!: string;

  @IsNotEmpty()
  @IsString()
  discordID!: string; // Include discordID

  @IsNotEmpty()
  response!: Response; // Use the Response enum
}
