// src/modules/api-gateway/api-gateway.service.ts
import { Injectable } from "@nestjs/common";
import { UserService, LeaderboardService, ScoreService } from "src/modules";
import { ActionHandler } from "./request-handlers/action.handler";

@Injectable()
export class ApiGatewayService {
  constructor(
    private readonly userService: UserService,
    private readonly scoreService: ScoreService,
    private readonly leaderboardService: LeaderboardService
  ) {}

  async handleRequest(
    discordID: string,
    action: string,
    requestData: any,
    req: any
  ) {
    try {
      console.log(`handleRequest called (reqId: ${req.id})`);

      // --- Add a custom property to the req object ---
      req.myCustomProperty = "test-value";

      const actionHandler = new ActionHandler(
        this.userService,
        this.leaderboardService,
        this.scoreService
      );

      console.log(`req passed to ActionHandler (reqId: ${req.id})`);
      const result = await actionHandler.handle({
        discordID,
        action,
        requestData,
        req, // Pass req explicitly to ActionHandler
      });

      return result;
    } catch (error) {
      console.error(`Error handling action ${action}:`, error);
      throw new Error(`Failed to handle action: ${action}`);
    }
  }
}
