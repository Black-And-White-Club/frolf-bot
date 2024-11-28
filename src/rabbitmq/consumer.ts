// src/amqp/consumer.ts

import * as amqp from "amqplib";

const connectionString = "amqp://your-rabbitmq-service:5672"; // Replace with your RabbitMQ service details

export async function consumeMessages(
  queue: string,
  callback: (message: any) => Promise<void>
) {
  try {
    const connection = await amqp.connect(connectionString);
    const channel = await connection.createChannel();
    await channel.assertQueue(queue, { durable: true });

    channel.consume(queue, async (message) => {
      if (message !== null) {
        const content = JSON.parse(message.content.toString());
        await callback(content);
        channel.ack(message);
      }
    });

    console.log("Consuming messages from queue:", queue);
  } catch (error) {
    console.error("Error consuming messages:", error);
    // Handle the error appropriately
  }
}
