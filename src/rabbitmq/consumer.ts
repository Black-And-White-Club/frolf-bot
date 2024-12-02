import { Injectable } from "@nestjs/common";
import { RabbitSubscribe } from "@golevelup/nestjs-rabbitmq";
import { LeaderboardService } from "../modules/leaderboard/leaderboard.service"; // Adjust the path if needed

@Injectable()
export class ConsumerService {
  constructor(private readonly leaderboardService: LeaderboardService) {} // Inject LeaderboardService

  @RabbitSubscribe({
    exchange: "main_exchange",
    routingKey: "queue.routing.key",
    queue: "example_queue",
  })
  async handleIncomingMessage(
    message: any,
    routingKey: string,
    queue: string,
    correlationId: string | null = null,
    resolve: ((value: any) => void) | null = null
  ) {
    try {
      console.log(`Message received on ${queue}:`, message);

      if (correlationId && resolve) {
        // This is a response to a request, resolve the Promise
        resolve({ tagExists: message.content.tagExists });
      } else {
        // Call the appropriate handler based on the routing key
        if (routingKey === "check-tag") {
          this.leaderboardService.handleCheckTagMessage(message);
        }
        // Add other routing key checks for different handlers as needed
      }
    } catch (error) {
      console.error("Error processing message:", error);
    }
  }
}
