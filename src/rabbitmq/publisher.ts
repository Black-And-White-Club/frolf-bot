// src/amqp/publisher.ts

import * as amqp from "amqplib";

const connectionString = process.env.RABBITMQ_URL || "amqp://localhost:5672";
const maxRetries = 3; // Maximum number of retries

export async function publishMessage(queue: string, message: any) {
  let retries = 0;
  while (retries < maxRetries) {
    try {
      const connection = await amqp.connect(connectionString);
      const channel = await connection.createChannel();

      await channel.assertQueue(queue, { durable: true });

      channel.sendToQueue(queue, Buffer.from(JSON.stringify(message)), {
        persistent: true,
      });

      console.log("Message sent to queue:", queue);

      await channel.close();
      await connection.close();

      return; // Message published successfully, exit the loop
    } catch (error) {
      console.error("Error publishing message:", error);
      retries++;
      console.log(`Retrying... Attempt ${retries} of ${maxRetries}`);
      // You can add a delay here before retrying (e.g., using setTimeout)
    }
  }

  console.error(`Failed to publish message after ${maxRetries} retries.`);
  // Handle the final failure appropriately (e.g., log to a service, alert, etc.)
}
