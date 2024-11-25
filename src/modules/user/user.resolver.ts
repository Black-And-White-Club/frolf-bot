import { Resolver, Query, Mutation, Args } from "@nestjs/graphql";
import { UserService } from "./user.service";
import { CreateUserDto } from "../../dto/user/create-user.dto";
import { UpdateUserDto } from "../../dto/user/update-user.dto";
import { UserRole } from "../../enums/user-role.enum";
import { validate } from "class-validator";
import { User } from "src/types.generated";

@Resolver("User")
export class UserResolver {
  constructor(private readonly userService: UserService) {
    console.log("UserResolver userService:", this.userService);
  }

  @Query(() => String, { nullable: true })
  async getUser(
    @Args("discordID", { nullable: true }) discordID?: string,
    @Args("tagNumber", { nullable: true }) tagNumber?: number
  ): Promise<User | null> {
    try {
      if (discordID) {
        const user = await this.userService.getUserByDiscordID(discordID);
        if (!user) {
          throw new Error(`User not found with discordID: ${discordID}`);
        }
        return user;
      } else if (tagNumber) {
        const user = await this.userService.getUserByTagNumber(tagNumber);
        if (!user) {
          throw new Error(`User not found with tagNumber: ${tagNumber}`);
        }
        return user;
      } else {
        throw new Error("Please provide either discordID or tagNumber");
      }
    } catch (error) {
      console.error("Error fetching user:", error);
      if (error instanceof Error) {
        // Type guard
        throw new Error(`Could not fetch user: ${error.message}`);
      } else {
        throw new Error(`Could not fetch user: ${error}`);
      }
    }
  }

  @Mutation(() => String)
  async createUser(@Args("input") input: CreateUserDto): Promise<User> {
    try {
      console.log("createUser called with input:", input);

      const errors = await validate(input);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }

      const user = await this.userService.createUser(input);

      console.log("User created:", user);
      return user;
    } catch (error) {
      console.error("Error creating user:", error);
      if (error instanceof Error) {
        // Type guard
        throw new Error(`Could not create user: ${error.message}`);
      } else {
        throw new Error(`Could not create user: ${error}`);
      }
    }
  }

  @Mutation(() => String)
  async updateUser(
    @Args("input") input: UpdateUserDto,
    @Args("requesterRole") requesterRole: UserRole
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
        // Type guard
        throw new Error(`Could not update user: ${error.message}`);
      } else {
        throw new Error(`Could not update user: ${error}`);
      }
    }
  }
}
