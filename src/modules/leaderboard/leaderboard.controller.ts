// src/modules/leaderboard/leaderboard.controller.ts
import {
  Controller,
  Get,
  Param,
  Put,
  Body,
  ParseIntPipe,
  Post,
  HttpException,
  HttpStatus,
} from "@nestjs/common"; // Import HttpException and HttpStatus
import { LeaderboardService } from "./leaderboard.service";
import { UpdateTagSource } from "src/enums";
import { ReceiveScoresDto } from "src/dto/leaderboard/receive-scores.dto";
import { plainToClass } from "class-transformer";
import { validate } from "class-validator";

@Controller("leaderboard")
export class LeaderboardController {
  constructor(private readonly leaderboardService: LeaderboardService) {}

  @Get()
  async getLeaderboard() {
    return this.leaderboardService.getLeaderboard();
  }

  @Get("users/:discordID/tag")
  async getUserTag(@Param("discordID") discordID: string) {
    try {
      // Assuming you want to fetch the tag with "manual" source
      const tag = await this.leaderboardService.getUserTag(
        discordID,
        0,
        UpdateTagSource.Manual
      );
      if (!tag) {
        throw new HttpException(
          "Tag not found for the provided discordID",
          HttpStatus.NOT_FOUND
        );
      }
      return tag;
    } catch (error) {
      if (error instanceof HttpException) {
        throw error;
      }
      console.error("Error fetching user tag:", error);
      throw new HttpException(
        "Failed to fetch user tag",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  @Put("users/:discordID/tag")
  async updateTag(
    @Param("discordID") discordID: string,
    @Body("tagNumber", ParseIntPipe) tagNumber: number
  ) {
    try {
      // Assuming the source is "manual" for this endpoint
      const source = UpdateTagSource.Manual;

      // You might want to move the DTO validation to a pipe for better separation of concerns
      const updateTagDto = plainToClass(ReceiveScoresDto, {
        discordID,
        tagNumber,
      });
      const errors = await validate(updateTagDto);
      if (errors.length > 0) {
        throw new HttpException("Validation failed!", HttpStatus.BAD_REQUEST);
      }

      await this.leaderboardService.initiateManualTagSwap(discordID, tagNumber); // Initiate the swap
      return { message: "Tag swap initiated." }; // Return a success message
    } catch (error) {
      if (error instanceof HttpException) {
        throw error;
      }
      console.error("Error updating tag:", error);
      throw new HttpException(
        "Failed to update tag",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }

  @Post("scores")
  async receiveScores(@Body() receiveScoresDto: ReceiveScoresDto) {
    try {
      const errors = await validate(receiveScoresDto);
      if (errors.length > 0) {
        throw new HttpException("Validation failed!", HttpStatus.BAD_REQUEST);
      }

      await this.leaderboardService.processScores(receiveScoresDto.scores);
      return this.leaderboardService.getLeaderboard();
    } catch (error) {
      if (error instanceof HttpException) {
        throw error;
      }
      console.error("Error receiving scores:", error);
      throw new HttpException(
        "Failed to receive scores",
        HttpStatus.INTERNAL_SERVER_ERROR
      );
    }
  }
}
