// src/amqp/publisher.ts

import * as amqp from "amqplib";

const connectionString = "amqp://your-rabbitmq-service:5672"; // Replace with your RabbitMQ service details

export async function publishMessage(queue: string, message: any) {
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
  } catch (error) {
    console.error("Error publishing message:", error);
    // Handle the error appropriately (e.g., retry, log, etc.)
  }
}
