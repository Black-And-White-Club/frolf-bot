// src/rabbitmq/publisher.ts

import { Injectable, Inject } from "@nestjs/common";
import { AmqpConnection } from "@golevelup/nestjs-rabbitmq";

@Injectable()
export class Publisher {
  constructor(
    @Inject("AMQP_CONNECTION") private readonly amqpConnection: AmqpConnection
  ) {}

  async publishMessage(queue: string, message: any) {
    const maxRetries = 3;
    let retries = 0;

    while (retries < maxRetries) {
      try {
        await this.amqpConnection.publish("", queue, message, {
          persistent: true,
        });

        console.log("Message sent to queue:", queue);
        return;
      } catch (error) {
        console.error("Error publishing message:", error);
        retries++;
        console.log(`Retrying... Attempt ${retries} of ${maxRetries}`);
      }
    }

    console.error(`Failed to publish message after ${maxRetries} retries.`);
  }

  async publishAndGetResponse(
    queue: string,
    message: any,
    responseQueue?: string,
    consumerName?: string
  ): Promise<any> {
    return new Promise(async (resolve, reject) => {
      try {
        const correlationId = Math.random().toString();

        const responseQueueName = responseQueue || "responses";
        const consumerNameToUse = consumerName || "responseConsumer";

        await this.amqpConnection.createSubscriber(
          async (responseMessage: any) => {
            if (responseMessage.properties.correlationId === correlationId) {
              resolve(JSON.parse(responseMessage.content.toString()));
            }
            return;
          },
          {
            exchange: "",
            queue: responseQueueName,
          },
          consumerNameToUse
        );

        await this.amqpConnection.publish("", queue, message, {
          persistent: true,
          correlationId,
          replyTo: responseQueueName,
        });
      } catch (error) {
        console.error("Error in publishAndGetResponse:", error);
        reject(error);
      }
    });
  }
}
