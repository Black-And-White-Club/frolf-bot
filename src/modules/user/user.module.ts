// src/modules/user/user.module.ts
import { Module } from "@nestjs/common";
import { UserController } from "./user.controller";
import { UserService } from "./user.service";
import { DatabaseModule } from "src/db/database.module";
import { users as UserModel } from "./user.model";
import { MessagingModule } from "src/rabbitmq/messaging.module"; // Import MessagingModule

@Module({
  imports: [
    DatabaseModule.forFeature(UserModel, "USER_DATABASE_CONNECTION"),
    MessagingModule, // Import MessagingModule here
  ],
  controllers: [UserController],
  providers: [UserService],
  exports: [UserService],
})
export class UserModule {}
