import { Injectable, Inject, OnModuleInit } from "@nestjs/common";
import { users as UserModel } from "./user.model"; // Ensure this import is correct
import { eq } from "drizzle-orm";
import { User as GraphQLUser } from "../../types.generated"; // Importing the GraphQL types
import { drizzle } from "drizzle-orm/node-postgres";
import { UserRole } from "../../enums/user-role.enum";

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
export class UserService implements OnModuleInit {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>
  ) {}

  // OnModuleInit ensures this runs after the module has been fully initialized
  onModuleInit() {
    console.log("User Service initialized");
    console.log("Injected DB instance:", this.db);
  }

  // Type guard to check if a string is a valid UserRole
  private isValidUserRole(role: any): role is UserRole {
    return Object.values(UserRole).includes(role);
  }

  async getUserByDiscordID(discordID: string): Promise<GraphQLUser | null> {
    try {
      const users = await this.db
        .select()
        .from(UserModel)
        .where(eq(UserModel.discordID, discordID))
        .execute();

      if (users.length > 0) {
        const user = users[0];
        return {
          __typename: "User", // Ensure this matches your GraphQL type
          discordID: user.discordID!,
          name: user.name!,
          tagNumber: user.tagNumber,
          role:
            user.role && this.isValidUserRole(user.role)
              ? user.role
              : "RATTLER", // Provide a default role instead of null
          createdAt: user.createdAt.toISOString(), // Include createdAt
          updatedAt: user.updatedAt.toISOString(), // Include updatedAt
        };
      }

      return null; // No user found
    } catch (error) {
      throw new Error("Failed to fetch user");
    }
  }

  async getUserByTagNumber(tagNumber: number): Promise<GraphQLUser | null> {
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
          throw new Error(`Invalid role for user with tag number ${tagNumber}`);
        }

        return {
          __typename: "User", // Ensure this matches your GraphQL type
          discordID: user.discordID!,
          name: user.name!,
          tagNumber: user.tagNumber!,
          role: user.role, // Return the valid role directly
          createdAt: user.createdAt.toISOString(), // Include createdAt
          updatedAt: user.updatedAt.toISOString(), // Include updatedAt
        };
      }

      return null; // No user found
    } catch (error) {
      throw new Error("Failed to fetch user");
    }
  }

  // Create user method
  async createUser(userData: UserData): Promise<GraphQLUser> {
    try {
      // Validate role
      if (!this.isValidUserRole(userData.role)) {
        throw new Error("Invalid user role");
      }

      // Construct the new user object with correct types
      const newUser = {
        name: userData.name,
        discordID: userData.discordID,
        tagNumber: userData.tagNumber || null,
        role: userData.role, // Use the role provided in userData
      };

      // Insert into the database
      const result = await this.db
        .insert(UserModel)
        .values(newUser) // Pass the constructed newUser object
        .returning();

      const insertedUser = result[0]; // Get the first inserted user

      // Transform the inserted user into the GraphQL User format
      return {
        __typename: "User", // Ensure this matches your GraphQL type
        discordID: insertedUser.discordID!,
        name: insertedUser.name!,
        tagNumber: insertedUser.tagNumber,
        role: insertedUser.role as UserRole, // Cast the role to UserRole
        createdAt: insertedUser.createdAt.toISOString(),
        updatedAt: insertedUser.updatedAt.toISOString(),
      };
    } catch (error) {
      // Handle error when type is unknown
      if (error instanceof Error) {
        throw new Error(`Failed to create user: ${error.message}`);
      }
      throw new Error("Failed to create user: Unknown error");
    }
  }

  // Update user function
  async updateUser(
    input: UpdateUserInput,
    requesterRole: UserRole
  ): Promise<GraphQLUser> {
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
          role: input.role !== undefined ? input.role : user.role, // Directly use the string enum value
        })
        .where(eq(UserModel.discordID, input.discordID))
        .execute();

      return {
        __typename: "User",
        discordID: user.discordID,
        name: input.name !== undefined ? input.name : user.name,
        tagNumber:
          input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
        role: input.role !== undefined ? input.role : user.role, // Directly use the string enum value
        createdAt: user.createdAt,
        updatedAt: new Date().toISOString(), // Use current timestamp for updatedAt
      };
    } catch (error) {
      if (error instanceof Error && error.message === "User not found") {
        throw new Error("User not found");
      }
      throw new Error("Failed to update user");
    }
  }
}
