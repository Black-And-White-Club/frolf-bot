// src/modules/user/user.service.ts

import { Inject, Injectable } from "@nestjs/common";
import { User } from "src/modules/user/user.entity";
import { Publisher } from "src/rabbitmq/publisher"; // Import the Publisher service
import { eq } from "drizzle-orm";
import { users as UserModel } from "./user.model";

@Injectable()
export class UserService {
  constructor(
    @Inject("USER_DATABASE_CONNECTION") private db: any,
    private readonly publisher: Publisher // Inject the Publisher service
  ) {}

  async getUserByDiscordID(discordID: string): Promise<User | null> {
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
    try {
      console.log("Creating user with data:", userData);

      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        role: userData.role,
        ...(userData.tagNumber && { tagNumber: userData.tagNumber }),
      };

      const result = await this.db
        .insert(UserModel)
        .values(newUser)
        .returning();
      const insertedUser = result[0];

      console.log("User created successfully:", insertedUser);

      // Publish RabbitMQ message for leaderboard update
      await this.publisher.publishMessage("userCreated", insertedUser);

      return insertedUser;
    } catch (error) {
      console.error("Error creating user:", error);
      throw error;
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

      // Publish RabbitMQ message for leaderboard update if role or tagNumber changed
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
