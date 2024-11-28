// src/modules/api-gateway/api-gateway.service.ts
import { Injectable } from "@nestjs/common";
import { UserService, LeaderboardService, ScoreService } from "src/modules";

@Injectable()
export class ApiGatewayService {
  constructor(
    private readonly userService: UserService,
    private readonly scoreService: ScoreService,
    private readonly leaderboardService: LeaderboardService
  ) {}

  async handleRequest(discordID: string, action: string, requestData: any) {
    const userRole = await this.userService.getUserByDiscordID(discordID)?.role;

    // Routing and authorization logic
    switch (action) {
      case "updateLeaderboard": {
        const enrichedRequest = { ...requestData, userRole };
        return this.leaderboardService.handleAction(enrichedRequest);
      }
      case "createUser": {
        return this.userService.createUser(requestData);
      }
      case "updateUser": {
        const { role, ...userInput } = requestData;
        return this.userService.updateUser(userInput, role);
      }
      case "updateScore": {
        const enrichedRequest = { ...requestData, userRole };
        return this.scoreService.handleAction(enrichedRequest);
      }
      // ... handle other actions and modules ...
      default:
        throw new Error(`Unknown action: ${action}`);
    }
  }
}
