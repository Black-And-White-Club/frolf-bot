// src/rabbitmq/messaging.module.ts
import { Module } from "@nestjs/common";
import { RabbitMQModule } from "@golevelup/nestjs-rabbitmq";
import { Publisher } from "./publisher";
import { ConsumerService } from "./consumer";
import { QueueService } from "./queue.service";
import { AmqpConnection } from "@golevelup/nestjs-rabbitmq"; // Import AmqpConnection

@Module({
  imports: [
    RabbitMQModule.forRoot(RabbitMQModule, {
      exchanges: [{ name: "main_exchange", type: "direct" }],
      uri: "amqp://localhost:5672",
      connectionInitOptions: { wait: false },
    }),
  ],
  providers: [
    Publisher,
    ConsumerService,
    QueueService,
    {
      provide: "AMQP_CONNECTION",
      useExisting: AmqpConnection,
    },
  ],
  exports: [Publisher, ConsumerService, QueueService, "AMQP_CONNECTION"], // Also export 'AMQP_CONNECTION'
})
export class MessagingModule {}
