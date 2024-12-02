// src/modules/user/user.service.ts

import { Inject, Injectable, OnModuleInit } from "@nestjs/common";
import { User } from "src/modules/user/user.entity";
import { Publisher } from "src/rabbitmq/publisher";
import { eq } from "drizzle-orm";
import { users as UserModel } from "./user.model";
import { ConsumerService } from "src/rabbitmq/consumer";
import { v4 as uuidv4 } from "uuid";

@Injectable()
export class UserService implements OnModuleInit {
  constructor(
    @Inject("USER_DATABASE_CONNECTION") private db: any,
    private readonly publisher: Publisher,
    private readonly consumerService: ConsumerService
  ) {
    console.log("UserService constructor called with db:", db);
    console.log("UserService constructor called");
    console.log("Publisher instance:", publisher);
    console.log("ConsumerService instance:", consumerService);
  }

  async onModuleInit() {
    // This will be called after the service is initialized
    console.log("UserService initialized");
  }

  async getUserByDiscordID(discordID: string): Promise<User | null> {
    console.log("getUserByDiscordID called with discordID:", discordID);
    try {
      const result = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.discordID, discordID));

      if (result.length > 0) {
        return result[0];
      }

      return null;
    } catch (error) {
      console.error("Error fetching user by Discord ID:", error);
      throw new Error("Failed to fetch user");
    }
  }

  async createUser(userData: User): Promise<User> {
    console.log("UserService.createUser called with:", userData);

    try {
      console.log("Creating user with data:", userData);

      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        role: userData.role,
        ...(userData.tagNumber && { tagNumber: userData.tagNumber }),
      };

      console.log("About to publish message to RabbitMQ");

      const correlationId = uuidv4();

      await this.publisher.publishMessage(
        "check-tag",
        {
          discordID: newUser.discordID,
          tagNumber: newUser.tagNumber,
        },
        { correlationId: correlationId }
      );

      const tagNumberCheckResult: { tagExists: boolean } = await new Promise(
        (resolve) => {
          this.consumerService.handleIncomingMessage(
            null,
            "check-tag-responses",
            "check-tag-responses-queue",
            correlationId,
            resolve
          );
        }
      );

      if (tagNumberCheckResult.tagExists) {
        console.error("Tag number already exists. User creation failed.");
        throw new Error("Tag number is already in use.");
      } else {
        console.log("Tag number is available");

        const result = await this.db
          .insert(UserModel)
          .values(newUser)
          .returning();
        const insertedUser = result[0];

        console.log("User created successfully:", insertedUser);

        await this.publisher.publishMessage("userCreated", insertedUser);

        return insertedUser;
      }
    } catch (error) {
      console.error("Error creating user:", error);
      if (error instanceof Error) {
        console.error("Error message:", error.message);
        console.error("Error stack trace:", error.stack);
      } else {
        console.error("Error object:", error);
      }
      return {} as User;
    }
  }

  async updateUser(discordID: string, updates: Partial<User>): Promise<User> {
    try {
      console.log(
        "Updating user with Discord ID:",
        discordID,
        "and updates:",
        updates
      );

      const result = await this.db
        .update(UserModel)
        .set(updates)
        .where(eq(UserModel.discordID, discordID))
        .returning();
      const updatedUser = result[0];

      console.log("User updated successfully:", updatedUser);

      if (updates.role || updates.tagNumber !== undefined) {
        await this.publisher.publishMessage("userUpdated", updatedUser);
      }

      return updatedUser;
    } catch (error) {
      console.error("Error updating user:", error);
      throw error;
    }
  }
}
