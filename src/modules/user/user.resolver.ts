// src/users/users.resolver.ts

import { Injectable } from "@nestjs/common";
import { UserService } from "./user.service";
import { CreateUserDto } from "src/dto/user/create-user.dto";
import { UpdateUserDto } from "src/dto/user/update-user.dto";
import { UserRole } from "src/enums/user-role.enum";
import { validate } from "class-validator";
import {
  User,
  CreateUserResponse,
  UpdateUserResponse,
} from "src/types.generated";
import { GraphQLContext } from "src/context/graphql-context.provider"; // Import the context

@Injectable()
export class UserResolver {
  constructor(
    private readonly userService: UserService,
    private readonly graphQLContext: GraphQLContext // Inject the context
  ) {}

  async getUser(reference: {
    discordID?: string;
    tagNumber?: number;
  }): Promise<User | null> {
    try {
      if (!reference.discordID && !reference.tagNumber) {
        throw new Error("Please provide either discordID or tagNumber");
      }

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
      }

      throw new Error("Please provide discordID");
    } catch (error) {
      console.error("Error fetching user:", error);
      if (error instanceof Error) {
        throw new Error(`Could not fetch user: ${error.message}`);
      } else {
        throw new Error(`Could not fetch user: ${error}`);
      }
    }
  }

  async createUser(input: CreateUserDto): Promise<CreateUserResponse> {
    console.log("Hey, createUser Resolver here.");
    try {
      console.log("Context in createUser resolver:", this.graphQLContext); // Access context here
      console.log("createUser resolver called with input:", input);
      const errors = await validate(input);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }

      // Ensure the discordID is a string
      const discordID = input.discordID.toString();

      const response = await this.userService.createUser({
        ...input,
        discordID, // Use the converted discordID
        tagNumber: input.tagNumber,
        createdAt: input.createdAt?.toISOString() ?? new Date().toISOString(),
        updatedAt: input.updatedAt?.toISOString() ?? new Date().toISOString(),
      });

      if (!response.success) {
        console.error("Failed to create user:", response.error);
        throw new Error(response.error || "Failed to create user.");
      }

      return response;
    } catch (error) {
      console.error("Error creating user:", error);
      if (error instanceof Error) {
        throw new Error(`Could not create user: ${error.message}`);
      } else {
        throw new Error(`Could not create user: ${error}`);
      }
    }
  }

  private isRoleUpdateAuthorized(
    input: UpdateUserDto,
    requesterRole: UserRole
  ): boolean {
    return (
      !input.role ||
      ![UserRole.ADMIN, UserRole.EDITOR].some((role) => input.role === role) ||
      requesterRole === UserRole.ADMIN
    );
  }

  async updateUser(
    input: UpdateUserDto,
    requesterRole: UserRole
  ): Promise<UpdateUserResponse> {
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

      if (!this.isRoleUpdateAuthorized(input, requesterRole)) {
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      const updatedUser = await this.userService.updateUser(
        {
          ...input,
          tagNumber: input.tagNumber, // Include tagNumber in the request
        },
        requesterRole
      );

      // Check if the updateUser method returned an error
      if (!updatedUser.success) {
        throw new Error(updatedUser.error || "Failed to update user.");
      }

      return updatedUser; // Return the UpdateUserResponse object
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
