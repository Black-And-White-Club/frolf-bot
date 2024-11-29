// src/modules/user/user.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { users as UserModel } from "src/schema";
import { eq } from "drizzle-orm";
import { UserRole } from "src/enums/user-role.enum";
import {
  User,
  CreateUserResponse,
  UpdateUserResponse,
} from "src/types.generated";
import { publishMessage } from "src/rabbitmq/publisher";

interface UpdateUserInput {
  discordID: string;
  name?: string;
  tagNumber?: number | null;
  role?: UserRole;
}

@Injectable()
export class UserService {
  constructor(@Inject("USER_DATABASE_CONNECTION") private db: any) {}

  private isValidUserRole(role: any): role is UserRole {
    return Object.values(UserRole).includes(role);
  }

  async getUserByDiscordID(discordID: string): Promise<User | null> {
    try {
      const result = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.discordID, discordID));

      if (result.length > 0) {
        const user = result[0];
        return {
          ...user,
          name: user.name ?? "",
          role: user.role as UserRole,
          createdAt: user.createdAt.toISOString(),
          updatedAt: user.updatedAt!.toISOString(),
          deletedAt: user.deletedAt ? user.deletedAt.toISOString() : undefined,
        };
      }

      return null;
    } catch (error) {
      console.error("Error fetching user by Discord ID:", error);
      throw new Error("Failed to fetch user");
    }
  }

  async createUser(userData: any): Promise<CreateUserResponse> {
    try {
      console.log("Creating user with data:", userData);
      console.log(`userService.createUser called (reqId: ${userData.req.id})`);

      if (userData.req.myCustomProperty) {
        console.log("myCustomProperty found:", userData.req.myCustomProperty);
      } else {
        console.log("myCustomProperty NOT found");
      }

      // Default to RATTLER role
      userData.role = UserRole.RATTLER as UserRole;

      if (!this.isValidUserRole(userData.role)) {
        throw new Error("Invalid user role");
      }

      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        role: userData.role,
        // Add tagNumber to newUser if provided
        ...(userData.tagNumber && { tagNumber: userData.tagNumber }),
      };

      const result = await this.db
        .insert(UserModel)
        .values(newUser)
        .returning();
      const insertedUser = result[0];

      console.log("User created successfully:", insertedUser);

      if (userData.tagNumber) {
        await publishMessage("tagNumberAssignmentRequest", {
          discordID: insertedUser.discordID,
          tagNumber: userData.tagNumber,
        });
      }

      return {
        success: true,
        user: {
          ...insertedUser,
          name: insertedUser.name ?? "",
          role: insertedUser.role as UserRole,
          createdAt: insertedUser.createdAt.toISOString(),
          updatedAt: insertedUser.updatedAt!.toISOString(),
          deletedAt: insertedUser.deletedAt
            ? insertedUser.deletedAt.toISOString()
            : undefined,
        },
      };
    } catch (error) {
      console.error("Error creating user:", error);

      if (error instanceof Error) {
        if (
          error.message.includes(
            "duplicate key value violates unique constraint"
          )
        ) {
          throw new Error("User with this Discord ID already exists.");
        } else {
          throw new Error(`Failed to create user: ${error.message}`);
        }
      } else {
        throw new Error("Failed to create user: Unknown error");
      }
    }
  }

  async updateUser(
    input: UpdateUserInput,
    requesterRole: UserRole
  ): Promise<UpdateUserResponse> {
    try {
      console.log("Updating user with data:", input);
      const user = await this.getUserByDiscordID(input.discordID);
      if (!user) {
        throw new Error("User not found");
      }

      // Early return for unauthorized role updates
      if (
        input.role &&
        (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
        requesterRole !== UserRole.ADMIN
      ) {
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      const updatedUser = await this.db
        .update(UserModel)
        .set({
          name: input.name !== undefined ? input.name : user.name,
          role: input.role !== undefined ? input.role : user.role,
        })
        .where(eq(UserModel.discordID, input.discordID))
        .returning()
        .execute();

      const returnedUser = updatedUser[0];

      // Publish a message to RabbitMQ for tagNumber update (if provided)
      if (input.tagNumber !== undefined) {
        await publishMessage("tagNumberUpdateRequest", {
          discordID: returnedUser.discordID,
          tagNumber: input.tagNumber,
        });
      }

      return {
        success: true,
        user: {
          ...returnedUser,
          name: returnedUser.name ?? "",
          role: returnedUser.role as UserRole,
          updatedAt: new Date().toISOString(),
          createdAt: returnedUser.createdAt.toISOString(),
          deletedAt: returnedUser.deletedAt
            ? returnedUser.deletedAt.toISOString()
            : undefined,
        },
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
