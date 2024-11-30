// src/rabbitmq/consumer.ts

import { Injectable, Inject } from "@nestjs/common";
import { AmqpConnection } from "@golevelup/nestjs-rabbitmq";

@Injectable()
export class Consumer {
  constructor(
    @Inject("AMQP_CONNECTION") private readonly amqpConnection: AmqpConnection
  ) {}

  async consumeMessages(
    queue: string,
    callback: (message: any) => Promise<void>,
    consumerName: string
  ) {
    try {
      await this.amqpConnection.createSubscriber(
        async (message: any) => {
          try {
            const content = JSON.parse(message.content.toString());
            await callback(content);
          } catch (error) {
            console.error("Error processing message:", error);
            // Handle the error appropriately
          }
        },
        {
          exchange: "",
          queue,
        },
        consumerName
      );

      console.log("Consuming messages from queue:", queue);
    } catch (error) {
      console.error("Error consuming messages:", error);
      // Handle the error appropriately
    }
  }
}
