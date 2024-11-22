import { Injectable } from "@nestjs/common";
import { db, User as UserModel } from "../db";
import { eq } from "drizzle-orm";
import { User as GraphQLUser, UserRole } from "../types.generated"; // Importing the GraphQL types
import { drizzle } from "drizzle-orm/node-postgres";

interface UpdateUserInput {
  discordID: string;
  name?: string; // Optional
  tagNumber?: number | null; // Optional
  role?: UserRole; // Optional
}

@Injectable()
export class UserService {
  constructor(private readonly db: ReturnType<typeof drizzle>) {
    console.log("Injected DB instance:", db);
  }
  async getUserByDiscordID(discordID: string): Promise<GraphQLUser | null> {
    const users = await db
      .select()
      .from(UserModel)
      .where(eq(UserModel.discordID, discordID))
      .execute();

    if (users.length > 0) {
      const user = users[0];

      return {
        __typename: "User", // Ensure this matches your GraphQL type
        discordID: user.discordID!, // Use discordID as the primary identifier
        name: user.name!, // Use non-null assertion if you're sure it exists
        tagNumber: user.tagNumber,
        role: user.role as UserRole,
      };
    }

    return null; // No user found
  }

  async getUserByTagNumber(tagNumber: number): Promise<GraphQLUser | null> {
    const users = await db
      .select()
      .from(UserModel)
      .where(eq(UserModel.tagNumber, tagNumber))
      .execute();

    if (users.length > 0) {
      const user = users[0];

      return {
        __typename: "User", // Ensure this matches your GraphQL type
        discordID: user.discordID!, // Use discordID as the primary identifier
        name: user.name!, // Use non-null assertion if you're sure it exists
        tagNumber: user.tagNumber,
        role: user.role as UserRole,
      };
    }

    return null; // No user found
  }

  async createUser(input: {
    name: string;
    discordID: string; // This should be a non-null string
    tagNumber?: number | null;
  }): Promise<GraphQLUser> {
    const existingUser = await this.getUserByDiscordID(input.discordID);
    if (existingUser) {
      throw new Error("User with this Discord ID already exists");
    }

    const newUser = {
      name: input.name,
      discordID: input.discordID,
      tagNumber: input.tagNumber || null,
      role: "RATTLER" as UserRole,
    };

    const result = await db
      .insert(UserModel)
      .values(newUser)
      .returning();
    const insertedUser = result[0];

    return {
      __typename: "User", // Ensure this matches your GraphQL type
      discordID: insertedUser.discordID, // This is guaranteed to be a non-null string
      name: newUser.name,
      tagNumber: newUser.tagNumber,
      role: newUser.role,
    };
  }

  async updateUser(
    input: UpdateUserInput,
    requesterRole: UserRole
  ): Promise<GraphQLUser> {
    const user = await this.getUserByDiscordID(input.discordID);
    if (!user) {
      throw new Error("User  not found");
    }

    if (
      input.role !== undefined &&
      (input.role === "ADMIN" || input.role === "EDITOR") &&
      requesterRole !== "ADMIN"
    ) {
      throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
    }

    await db
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
      __typename: "User", // Ensure this matches your GraphQL type
      discordID: user.discordID, // Use discordID as the primary identifier
      name: input.name !== undefined ? input.name : user.name,
      tagNumber:
        input.tagNumber !== undefined ? input.tagNumber : user.tagNumber,
      role: input.role !== undefined ? input.role : user.role,
    };
  }
}
