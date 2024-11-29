import * as amqp from "amqplib";

const connectionString = process.env.RABBITMQ_URL || "amqp://localhost:5672"; // Use environment variable or default

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
        try {
          const content = JSON.parse(message.content.toString());
          await callback(content);
        } catch (error) {
          console.error("Error processing message:", error);
          // Handle the error appropriately, e.g., requeue the message or send it to a dead-letter queue
        } finally {
          channel.ack(message);
        }
      }
    });

    console.log("Consuming messages from queue:", queue);
  } catch (error) {
    console.error("Error consuming messages:", error);
    // Handle the error appropriately, e.g., retry the connection
  }
}
