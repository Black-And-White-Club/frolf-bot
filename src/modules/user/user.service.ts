import { Injectable, Inject } from "@nestjs/common";
import { users as UserModel } from "./user.model"; // Ensure this import is correct
import { eq } from "drizzle-orm";
import { User as GraphQLUser } from "../../types.generated"; // Importing the GraphQL types
import { drizzle } from "drizzle-orm/node-postgres";
import { UserRole } from "../../enums/user-role.enum";
import { LoggingService } from "../../utils/logger"; // Import the LoggingService

interface UpdateUserInput {
  discordID: string;
  name?: string; // Optional
  tagNumber?: number | null; // Optional
  role?: UserRole; // Optional
}

interface UserData {
  name: string;
  discordID: string;
  tagNumber: number | null;
  role: UserRole; // This should be a valid UserRole string
}

@Injectable()
export class UserService {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>,
    private readonly loggingService: LoggingService // Inject LoggingService
  ) {
    console.log("Injected DB instance:", db);
    this.loggingService.logInfo("User Service initialized");
  }

  // Type guard to check if a string is a valid UserRole
  private isValidUserRole(role: any): role is UserRole {
    return Object.values(UserRole).includes(role);
  }

  async getUserByDiscordID(discordID: string): Promise<GraphQLUser | null> {
    this.loggingService.logInfo(`Fetching user by discordID: ${discordID}`);
    try {
      const users = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.discordID, discordID))
        .execute();

      if (users.length > 0) {
        const user = users[0];
        this.loggingService.logInfo(`User  found: ${JSON.stringify(user)}`);
        return {
          __typename: "User", // Ensure this matches your GraphQL type
          discordID: user.discordID!,
          name: user.name!,
          tagNumber: user.tagNumber,
          role:
            user.role && this.isValidUserRole(user.role)
              ? user.role
              : "RATTLER", // Provide a default role instead of null
        };
      }

      this.loggingService.logWarn(`User not found for discordID: ${discordID}`);
      return null; // No user found
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      this.loggingService.logError(
        `Error fetching user by discordID: ${errorMessage}`
      );
      throw new Error("Failed to fetch user");
    }
  }

  async getUserByTagNumber(tagNumber: number): Promise<GraphQLUser | null> {
    this.loggingService.logInfo(`Fetching user by tagNumber: ${tagNumber}`);
    try {
      const users = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.tagNumber, tagNumber))
        .execute();

      if (users.length > 0) {
        const user = users[0];

        // Ensure the role is valid before returning
        if (!this.isValidUserRole(user.role)) {
          this.loggingService.logError(
            `Invalid role for user with tag number ${tagNumber}`
          );
          throw new Error(`Invalid role for user with tag number ${tagNumber}`);
        }

        this.loggingService.logInfo(`User found: ${JSON.stringify(user)}`);
        return {
          __typename: "User", // Ensure this matches your GraphQL type
          discordID: user.discordID!,
          name: user.name!,
          tagNumber: user.tagNumber,
          role: user.role, // Return the valid role directly
        };
      }

      this.loggingService.logWarn(`User not found for tagNumber: ${tagNumber}`);
      return null; // No user found
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      this.loggingService.logError(
        `Error fetching user by tagNumber: ${errorMessage}`
      );
      throw new Error("Failed to fetch user");
    }
  }

  async createUser(userData: UserData) {
    this.loggingService.logInfo(
      `Creating user with data: ${JSON.stringify(userData)}`
    );
    try {
      // Construct the new user object with correct types
      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        tagNumber: userData.tagNumber,
        role: "RATTLER", // Cast to string if necessary
      };

      // Insert into the database
      const result = await this.db
        .insert(UserModel)
        .values(newUser) // Pass the constructed newUser  object
        .returning();

      const insertedUser = result[0]; // Get the first inserted user
      this.loggingService.logInfo(
        `User created successfully: ${JSON.stringify(insertedUser)}`
      );
      return insertedUser; // Return the inserted user
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      this.loggingService.logError(`Error creating user: ${errorMessage}`);
      throw new Error("Failed to create user");
    }
  }

  // Update user function
  async updateUser(
    input: UpdateUserInput,
    requesterRole: UserRole
  ): Promise<GraphQLUser> {
    this.loggingService.logInfo(
      `Updating user with discordID: ${input.discordID}`
    );
    try {
      const user = await this.getUserByDiscordID(input.discordID);
      if (!user) {
        this.loggingService.logWarn(
          `User not found for update: ${input.discordID}`
        );
        throw new Error("User not found");
      }

      if (
        input.role !== undefined &&
        (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
        requesterRole !== UserRole.ADMIN
      ) {
        this.loggingService.logError(
          `Unauthorized role change attempt by ${requesterRole}`
        );
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      await this.db
        .update(UserModel)
        .set({
          name: input.name !== undefined ? input.name : user.name,
          tagNumber:
            input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
          role: input.role !== undefined ? input.role : user.role, // Directly use the string enum value
        })
        .where(eq(UserModel.discordID, input.discordID))
        .execute();

      this.loggingService.logInfo(
        `User updated successfully: ${input.discordID}`
      );
      return {
        discordID: user.discordID,
        name: input.name !== undefined ? input.name : user.name,
        tagNumber:
          input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
        role: input.role !== undefined ? input.role : user.role, // Directly use the string enum value
      };
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      this.loggingService.logError(`Error updating user: ${errorMessage}`);
      throw new Error("Failed to update user");
    }
  }
}
