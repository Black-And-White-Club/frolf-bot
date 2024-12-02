import { Injectable } from "@nestjs/common";
import { AmqpConnection } from "@golevelup/nestjs-rabbitmq";
import { v4 as uuidv4 } from "uuid";

@Injectable()
export class Publisher {
  constructor(private readonly amqpConnection: AmqpConnection) {}

  async publishMessage(
    routingKey: string,
    message: any,
    options?: { correlationId?: string }
  ): Promise<void> {
    try {
      await this.amqpConnection.publish(
        "main_exchange",
        routingKey,
        message,
        options
      );
    } catch (error) {
      console.error("Error publishing message:", error);
    }
  }

  private generateCorrelationId(): string {
    return uuidv4();
  }

  async publishAndGetResponse<T>(
    exchange: string,
    routingKey: string,
    message: any
  ): Promise<{ tagExists: boolean }> {
    return new Promise((resolve, reject) => {
      const correlationId = this.generateCorrelationId();

      const queue = "response_queue";
      const timeout = setTimeout(() => {
        reject(new Error("Timeout waiting for response"));
      }, 10000);

      this.amqpConnection.publish(exchange, routingKey, message, {
        correlationId,
        replyTo: queue,
      });

      type SubscribeResponse = number;

      this.amqpConnection.createSubscriber(
        async (response: any): Promise<void> => {
          return new Promise<void>((resolve, reject) => {
            try {
              if (response.properties.correlationId === correlationId) {
                clearTimeout(timeout);

                const tagNumber =
                  typeof response.content === "number"
                    ? response.content
                    : undefined;

                if (tagNumber === undefined) {
                  reject(
                    new Error("Invalid response content: expected number")
                  );
                } else {
                  resolve();
                }
              }
            } catch (error) {
              reject(error);
            }
          });
        },
        { queue },
        "roundTagConsumerHandler",
        {}
      );
    });
  }
}
