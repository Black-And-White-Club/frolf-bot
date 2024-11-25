import { Injectable } from "@nestjs/common";
import { users as UserModel } from "./user.model";
import { eq } from "drizzle-orm";
import { User } from "../../types.generated";
import { drizzle } from "drizzle-orm/node-postgres";
import { UserRole } from "../../enums/user-role.enum";
import { createDbClient } from "../../db";

interface UpdateUserInput {
  discordID: string;
  name?: string;
  tagNumber?: number | null;
  role?: UserRole;
}

interface UserData {
  name: string;
  discordID: string;
  tagNumber: number | null;
  role: UserRole;
}

@Injectable()
export class UserService {
  private readonly db: ReturnType<typeof drizzle>;

  constructor(db: ReturnType<typeof drizzle>) {
    // Accept db as a parameter
    this.db = db;
    console.log("UserService initialized");
    console.log("Database connection established");
  }

  private isValidUserRole(role: any): role is UserRole {
    return Object.values(UserRole).includes(role);
  }

  async getUserByDiscordID(discordID: string): Promise<User | null> {
    try {
      const users = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.discordID, discordID))
        .execute();

      if (users.length > 0) {
        const user = users[0];
        return {
          __typename: "User",
          discordID: user.discordID,
          name: user.name!,
          tagNumber: user.tagNumber,
          role: user.role as UserRole,
          createdAt: user.createdAt.toISOString(),
          updatedAt: user.updatedAt.toISOString(),
        };
      }

      return null;
    } catch (error) {
      console.error("Error fetching user by Discord ID:", error);
      throw new Error("Failed to fetch user");
    }
  }

  async getUserByTagNumber(tagNumber: number): Promise<User | null> {
    try {
      const users = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.tagNumber, tagNumber))
        .execute();

      if (users.length > 0) {
        const user = users[0];

        if (!this.isValidUserRole(user.role)) {
          throw new Error(`Invalid role for user with tag number ${tagNumber}`);
        }

        return {
          __typename: "User",
          discordID: user.discordID,
          name: user.name!,
          tagNumber: user.tagNumber,
          role: user.role,
          createdAt: user.createdAt.toISOString(),
          updatedAt: user.updatedAt.toISOString(),
        };
      }

      return null;
    } catch (error) {
      console.error("Error fetching user by tag number:", error);
      throw new Error("Failed to fetch user");
    }
  }

  async createUser(userData: UserData): Promise<User> {
    try {
      if (!this.isValidUserRole(userData.role)) {
        throw new Error("Invalid user role");
      }

      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        tagNumber: userData.tagNumber || null,
        role: userData.role,
      };

      const result = await this.db
        .insert(UserModel)
        .values(newUser)
        .returning();

      const insertedUser = result[0];

      return {
        __typename: "User",
        discordID: insertedUser.discordID,
        name: insertedUser.name!,
        tagNumber: insertedUser.tagNumber,
        role: insertedUser.role as UserRole,
        createdAt: insertedUser.createdAt.toISOString(),
        updatedAt: insertedUser.updatedAt.toISOString(),
      };
    } catch (error) {
      console.error("Error creating user:", error);
      if (error instanceof Error) {
        throw new Error(`Failed to create user: ${error.message}`);
      }
      throw new Error("Failed to create user: Unknown error");
    }
  }

  async updateUser(
    input: UpdateUserInput,
    requesterRole: UserRole
  ): Promise<User> {
    try {
      const user = await this.getUserByDiscordID(input.discordID);
      if (!user) {
        throw new Error("User not found");
      }

      if (
        input.role !== undefined &&
        (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
        requesterRole !== UserRole.ADMIN
      ) {
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      await this.db
        .update(UserModel)
        .set({
          name: input.name !== undefined ? input.name : user.name,
          tagNumber:
            input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
          role: input.role !== undefined ? input.role : user.role,
        })
        .where(eq(UserModel.discordID, input.discordID))
        .execute();

      return {
        __typename: "User",
        discordID: user.discordID,
        name: input.name !== undefined ? input.name : user.name,
        tagNumber:
          input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
        role: input.role !== undefined ? input.role : user.role,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
    } catch (error) {
      console.error("Error updating user:", error);
      if (error instanceof Error && error.message === "User not found") {
        throw new Error("User not found");
      }
      throw new Error("Failed to update user");
    }
  }
}
