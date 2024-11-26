// src/users/users.resolver.ts
import { Injectable } from "@nestjs/common";
import { UserService } from "./user.service";
import { CreateUserDto } from "src/dto/user/create-user.dto";
import { UpdateUserDto } from "src/dto/user/update-user.dto";
import { UserRole } from "../../enums/user-role.enum";
import { validate } from "class-validator";
import { User } from "src/types.generated";

@Injectable()
export class UserResolver {
  constructor(private readonly userService: UserService) {}

  async getUser(reference: {
    discordID?: string;
    tagNumber?: number;
  }): Promise<User | null> {
    try {
      if (reference.discordID) {
        const user = await this.userService.getUserByDiscordID(
          reference.discordID
        );
        if (!user) {
          throw new Error(
            `User not found with discordID: ${reference.discordID}`
          );
        }
        return user;
      } else if (reference.tagNumber) {
        const user = await this.userService.getUserByTagNumber(
          reference.tagNumber
        );
        if (!user) {
          throw new Error(
            `User not found with tagNumber: ${reference.tagNumber}`
          );
        }
        return user;
      } else {
        throw new Error("Please provide either discordID or tagNumber");
      }
    } catch (error) {
      console.error("Error fetching user:", error);
      if (error instanceof Error) {
        throw new Error(`Could not fetch user: ${error.message}`);
      } else {
        throw new Error(`Could not fetch user: ${error}`);
      }
    }
  }

  async createUser(input: CreateUserDto): Promise<User> {
    try {
      const errors = await validate(input);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }

      const user = await this.userService.createUser({
        ...input,
        tagNumber: input.tagNumber ?? undefined,
        createdAt: input.createdAt
          ? input.createdAt.toISOString()
          : new Date().toISOString(), // Convert or use current date
        updatedAt: input.updatedAt?.toISOString() ?? new Date().toISOString(), // Convert or use current date
      });
      return user;
    } catch (error) {
      console.error("Error creating user:", error);
      if (error instanceof Error) {
        throw new Error(`Could not create user: ${error.message}`);
      } else {
        throw new Error(`Could not create user: ${error}`);
      }
    }
  }

  async updateUser(
    input: UpdateUserDto,
    requesterRole: UserRole
  ): Promise<User> {
    try {
      const errors = await validate(input);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }

      if (!input.discordID) {
        throw new Error("Discord ID is required");
      }

      const currentUser = await this.userService.getUserByDiscordID(
        input.discordID
      );
      if (!currentUser) {
        throw new Error("User not found");
      }

      if (
        input.role &&
        (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
        requesterRole !== UserRole.ADMIN
      ) {
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      const updatedUser = await this.userService.updateUser(
        input,
        requesterRole
      );
      return updatedUser;
    } catch (error) {
      console.error("Error updating user:", error);
      if (error instanceof Error) {
        throw new Error(`Could not update user: ${error.message}`);
      } else {
        throw new Error(`Could not update user: ${error}`);
      }
    }
  }
}
