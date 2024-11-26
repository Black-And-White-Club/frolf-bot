// src/modules/user/user.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { users as UserModel } from "../../schema";
import { eq } from "drizzle-orm";
import { UserRole } from "../../enums/user-role.enum";
import { User } from "src/types.generated";

interface UpdateUserInput {
  discordID: string;
  name?: string;
  tagNumber?: number | null;
  role?: UserRole;
}

@Injectable()
export class UserService {
  constructor(@Inject("USER_DATABASE_CONNECTION") private db: any) {} // Inject module-specific db connection

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
          tagNumber: user.tagNumber ?? undefined,
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
      const result = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.tagNumber, tagNumber));

      if (result.length > 0) {
        const user = result[0];

        if (!this.isValidUserRole(user.role)) {
          throw new Error(`Invalid role for user with tag number ${tagNumber}`);
        }

        return {
          ...user,
          name: user.name ?? "",
          role: user.role as UserRole,
          createdAt: user.createdAt.toISOString(),
          updatedAt: user.updatedAt!.toISOString(),
          deletedAt: user.deletedAt ? user.deletedAt.toISOString() : undefined,
          tagNumber: user.tagNumber ?? undefined,
        };
      }

      return null;
    } catch (error) {
      console.error("Error fetching user by tag number:", error);
      throw new Error("Failed to fetch user");
    }
  }

  async createUser(userData: User): Promise<User> {
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
        ...insertedUser,
        name: insertedUser.name ?? "",
        role: insertedUser.role as UserRole,
        createdAt: insertedUser.createdAt.toISOString(),
        updatedAt: insertedUser.updatedAt!.toISOString(),
        deletedAt: insertedUser.deletedAt
          ? insertedUser.deletedAt.toISOString()
          : undefined,
        tagNumber: insertedUser.tagNumber ?? undefined,
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

      const updatedUser = await this.db
        .update(UserModel)
        .set({
          name: input.name !== undefined ? input.name : user.name,
          tagNumber:
            input.tagNumber !== undefined
              ? input.tagNumber
              : user.tagNumber ?? undefined,
          role: input.role !== undefined ? input.role : user.role,
        })
        .where(eq(UserModel.discordID, input.discordID))
        .returning()
        .execute();

      const returnedUser = updatedUser[0];

      return {
        ...returnedUser,
        name: returnedUser.name ?? "",
        tagNumber: returnedUser.tagNumber ?? undefined,
        role: returnedUser.role as UserRole,
        updatedAt: new Date().toISOString(),
        createdAt: returnedUser.createdAt.toISOString(),
        deletedAt: returnedUser.deletedAt
          ? returnedUser.deletedAt.toISOString()
          : undefined,
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
