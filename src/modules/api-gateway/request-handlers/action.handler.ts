// src/modules/api-gateway/request-handlers/action.handler.ts
import { RequestHandler } from "./request-handler.interface";
import { UserService, LeaderboardService, ScoreService } from "src/modules";

export class ActionHandler implements RequestHandler {
  private nextHandler: RequestHandler | null = null;

  constructor(
    private readonly userService: UserService,
    private readonly leaderboardService: LeaderboardService,
    private readonly scoreService: ScoreService
  ) {}

  setNext(handler: RequestHandler): this {
    this.nextHandler = handler;
    return this;
  }

  async handle(request: any): Promise<any> {
    const { discordID, action, requestData, reqId } = request;
    console.log(`ActionHandler.handle called (reqId: ${reqId})`); // Log req.id

    const user = await this.userService.getUserByDiscordID(discordID);
    const userRole = user?.role;

    const actionMap: { [key: string]: { service: any; method: string } } = {
      updateLeaderboard: {
        service: this.leaderboardService,
        method: "processScores",
      },
      createUser: { service: this.userService, method: "createUser" },
      updateUser: { service: this.userService, method: "updateUser" },
      updateScore: { service: this.scoreService, method: "handleAction" },
    };

    try {
      const { service, method } = actionMap[action];

      console.log("Identified service:", service);
      console.log("Identified method:", method);

      if (!service || !method) {
        throw new Error(`Unknown action: ${action}`);
      }

      let result;
      if (method === "updateUser") {
        const { role, ...userInput } = requestData;
        result = await service[method](userInput, role);
      } else if (method === "createUser") {
        // Sanitize createUser input
        const sanitizedInput = {
          name: requestData.name,
          discordID: requestData.discordID,
          tagNumber: requestData.tagNumber, // Include tagNumber if needed
        };
        console.log(`req passed to userService.createUser (reqId: ${reqId})`);
        result = await service[method](sanitizedInput);
      } else {
        const enrichedRequest = { ...requestData, userRole };
        result = await service[method](enrichedRequest);
      }

      if (this.nextHandler) {
        return this.nextHandler.handle({ ...request, result });
      }

      return result;
    } catch (error) {
      console.error(`Error handling action ${action}:`, error);
      throw new Error(`Failed to handle action: ${action}`);
    }
  }
}
