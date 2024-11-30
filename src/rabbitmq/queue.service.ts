// src/rabbitmq/queue.service.ts

import { Injectable, Inject } from "@nestjs/common";
import { AmqpConnection } from "@golevelup/nestjs-rabbitmq";
import { LeaderboardService } from "src/modules/leaderboard/leaderboard.service";
import { UpdateTagSource } from "src/enums";

@Injectable()
export class QueueService {
  private activeSwapId: string | null = null;

  constructor(
    private readonly leaderboardService: LeaderboardService,
    @Inject("AMQP_CONNECTION") private readonly amqpConnection: AmqpConnection
  ) {}

  async getActiveSwapId(): Promise<string | null> {
    return this.activeSwapId;
  }

  async processTagSwapRequest(
    queueGroupName: string,
    source: UpdateTagSource
  ): Promise<{
    success: boolean;
    successfulSwaps?: { discordID: string; tagNumber: number }[];
    unmatchedUsers?: string[];
  }> {
    return new Promise<{
      success: boolean;
      successfulSwaps?: { discordID: string; tagNumber: number }[];
      unmatchedUsers?: string[];
    }>((resolve) => {
      const tagRequests: { [tagNumber: number]: string[] } = {};
      const matchedSwaps: { discordID: string; tagNumber: number }[] = [];

      const intervalId = setInterval(async () => {
        const message = await this.amqpConnection.channel.get(queueGroupName, {
          noAck: false,
        });

        if (message) {
          const { discordID, tagNumber } = JSON.parse(
            message.content.toString()
          );

          if (!tagRequests[tagNumber]) {
            tagRequests[tagNumber] = [];
          }
          tagRequests[tagNumber].push(discordID);

          if (tagRequests[tagNumber].length >= 2) {
            const [discordID1, discordID2] = tagRequests[tagNumber];
            matchedSwaps.push({ discordID: discordID1, tagNumber });
            matchedSwaps.push({ discordID: discordID2, tagNumber });

            tagRequests[tagNumber] = tagRequests[tagNumber].filter(
              (id) => id !== discordID1 && id !== discordID2
            );

            try {
              await this.leaderboardService.updateTag(
                [
                  { discordID: discordID1, tagNumber: tagNumber },
                  { discordID: discordID2, tagNumber: tagNumber },
                ],
                source
              );
            } catch (error) {
              console.error("Failed to update tags:", error);
              // Consider adding retry logic or other error handling here
            }
          }

          this.amqpConnection.channel.ack(message);
        }

        const queueInfo = await this.amqpConnection.channel.checkQueue(
          queueGroupName
        );
        const timeout = Date.now() + 180000; // 3 minutes timeout
        if (queueInfo.messageCount === 0 || Date.now() > timeout) {
          clearInterval(intervalId);
          this.activeSwapId = null;

          if (matchedSwaps.length > 0) {
            resolve({ success: true, successfulSwaps: matchedSwaps });
          } else {
            const unmatchedUsers = Object.values(tagRequests).flat();
            resolve({ success: false, unmatchedUsers });
          }
        }
      }, 1000);
    });
  }

  async publishMessage(queueName: string, message: any): Promise<void> {
    await this.amqpConnection.publish("", queueName, message, {
      persistent: true,
    });
  }
}
