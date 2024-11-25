import { Resolver, Query, Mutation, Args, Context } from "@nestjs/graphql";
import { UserService } from "./user.service";
import { CreateUserDto } from "../../dto/user/create-user.dto";
import { UpdateUserDto } from "../../dto/user/update-user.dto";
import { UserRole } from "../../enums/user-role.enum";
import { validate } from "class-validator";
import { GraphQLResolveInfo } from "graphql";

@Resolver()
export class UserResolver {
  static Mutation: any;
  static Query: any;
  constructor(
    private readonly userService: UserService // Access instance-specific service (userService)
  ) {}

  @Query(() => String)
  async getUser(
    @Args("discordID", { nullable: true }) discordID?: string,
    @Args("tagNumber", { nullable: true }) tagNumber?: number,
    info?: GraphQLResolveInfo
  ): Promise<any> {
    try {
      let user: any = null;

      if (discordID) {
        user = await this.userService.getUserByDiscordID(discordID);
      } else if (tagNumber) {
        user = await this.userService.getUserByTagNumber(tagNumber);
      } else {
        throw new Error("Please provide either discordID or tagNumber");
      }

      if (!user) {
        throw new Error(`User not found: ${discordID || tagNumber}`);
      }

      return user;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : JSON.stringify(error);
      throw new Error(`Could not fetch user: ${errorMessage}`);
    }
  }

  @Mutation(() => String)
  async createUser(@Args("input") input: CreateUserDto): Promise<any> {
    try {
      const existingUser = await this.userService.getUserByDiscordID(
        input.discordID
      );
      if (existingUser) {
        throw new Error("User with this Discord ID already exists.");
      }

      if (input.tagNumber !== null && input.tagNumber !== undefined) {
        const existingTagUser = await this.userService.getUserByTagNumber(
          input.tagNumber
        );
        if (existingTagUser) {
          throw new Error("User with this tag number already exists.");
        }
      }

      const userDto = new CreateUserDto();
      Object.assign(userDto, input, {
        createdAt: new Date(),
        updatedAt: new Date(),
      });

      const errors = await validate(userDto);
      if (errors.length > 0) {
        throw new Error("Validation failed: " + JSON.stringify(errors));
      }

      const user = await this.userService.createUser(userDto);
      return user;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      throw new Error("Could not create user: " + errorMessage);
    }
  }

  @Mutation(() => String)
  async updateUser(
    @Args("input") input: UpdateUserDto,
    @Args("requesterRole") requesterRole: UserRole
  ): Promise<any> {
    // Check if discordID is provided
    if (!input.discordID) {
      throw new Error("Discord ID is required");
    }

    try {
      const currentUser = await this.userService.getUserByDiscordID(
        input.discordID
      );

      if (!currentUser) {
        throw new Error("User not found");
      }

      const updateUserDto: UpdateUserDto = {
        discordID: input.discordID,
        role: input.role ?? currentUser.role,
        name: input.name ?? currentUser.name,
        tagNumber:
          input.tagNumber === undefined
            ? currentUser.tagNumber
            : input.tagNumber,
      };

      const errors = await validate(updateUserDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      if (
        input.role &&
        (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
        requesterRole !== UserRole.ADMIN
      ) {
        throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
      }

      const updatedUser = await this.userService.updateUser(
        updateUserDto,
        requesterRole
      );
      return updatedUser;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Unknown error occurred";
      throw new Error("Could not update user: " + errorMessage);
    }
  }
}
