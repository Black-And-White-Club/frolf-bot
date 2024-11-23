import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { UserInput } from "../../types.generated"; // Assuming you have defined UserInput somewhere
import { UserRole } from "../../enums/user-role.enum";
import { UserService } from "./user.service";
import { CreateUserDto } from "../../dto/user/create-user.dto";
import { UpdateUserDto } from "../../dto/user/update-user.dto";
import { LoggingService } from "../../utils/logger"; // Import the LoggingService

export const UserResolver = {
  Query: {
    async getUser(
      _: any,
      args: { discordID?: string; tagNumber?: number },
      context: { userService: UserService; loggingService: LoggingService } // Add loggingService to context
    ) {
      context.loggingService.logInfo(
        `Fetching user with args: ${JSON.stringify(args)}`
      );
      try {
        if (args.discordID) {
          const user = await context.userService.getUserByDiscordID(
            args.discordID
          );
          if (!user) {
            context.loggingService.logWarn(
              `User  not found by discordID: ${args.discordID}`
            );
            throw new Error("User  not found by discordID");
          }
          return user;
        } else if (args.tagNumber) {
          const user = await context.userService.getUserByTagNumber(
            args.tagNumber
          );
          if (!user) {
            context.loggingService.logWarn(
              `User not found by tagNumber: ${args.tagNumber}`
            );
            throw new Error("User not found by Tag");
          }
          return user;
        } else {
          context.loggingService.logError(
            "No identifier provided for user lookup"
          );
          throw new Error("Please provide either discordID or tagNumber");
        }
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : "Unknown error occurred";
        context.loggingService.logError(
          `Error in getUser  resolver: ${errorMessage}`
        );
        throw new Error("Could not fetch user: " + errorMessage);
      }
    },
  },
  Mutation: {
    async createUser(
      _: any,
      { input }: { input: CreateUserDto },
      {
        userService,
        loggingService,
      }: { userService: UserService; loggingService: LoggingService }
    ) {
      loggingService.logInfo(
        `Creating user with input: ${JSON.stringify(input)}`
      );
      try {
        // Check if the user already exists
        const existingUser = await userService.getUserByDiscordID(
          input.discordID
        );
        if (existingUser) {
          loggingService.logWarn(
            `User  with Discord ID already exists: ${input.discordID}`
          );
          throw new Error("User  with this Discord ID already exists.");
        }

        const userDto: CreateUserDto = {
          discordID: input.discordID,
          name: input.name,
          tagNumber: input.tagNumber ?? null,
          role: input.role, // Ensure this is a valid role
        };

        const errors = await validate(userDto);
        if (errors.length > 0) {
          loggingService.logError("Validation failed for user creation");
          throw new Error("Validation failed!");
        }

        const user = await userService.createUser(userDto);
        loggingService.logInfo(
          `User  created successfully: ${JSON.stringify(user)}`
        );
        return user;
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : "Unknown error occurred";
        loggingService.logError(
          `Error in createUser  resolver: ${errorMessage}`
        );
        throw new Error("Could not create user: " + errorMessage); // Provide specific error messages
      }
    },

    async updateUser(
      _: any,
      args: { input: UpdateUserDto; requesterRole: UserRole },
      context: { userService: UserService; loggingService: LoggingService }
    ) {
      const { input, requesterRole } = args;
      context.loggingService.logInfo(
        `Updating user with input: ${JSON.stringify(input)}`
      );

      try {
        const currentUser = await context.userService.getUserByDiscordID(
          input.discordID
        );
        if (!currentUser) {
          context.loggingService.logWarn(
            `User  not found for update: ${input.discordID}`
          );
          throw new Error("User  not found");
        }

        // Prepare the update DTO
        const updateUserDto: UpdateUserDto = {
          discordID: input.discordID || currentUser.discordID,
          role: input.role || currentUser.role,
          name: input.name || currentUser.name,
          tagNumber:
            input.tagNumber === undefined
              ? currentUser.tagNumber
              : input.tagNumber,
        };

        // Validate the update DTO
        const errors = await validate(updateUserDto);
        if (errors.length > 0) {
          context.loggingService.logError("Validation failed for user update");
          throw new Error("Validation failed!");
        }

        // Check permissions for role updates
        if (
          input.role &&
          (input.role === UserRole.ADMIN || input.role === UserRole.EDITOR) &&
          requesterRole !== UserRole.ADMIN
        ) {
          context.loggingService.logError(
            `Unauthorized role change attempt by ${requesterRole}`
          );
          throw new Error("Only ADMIN can change roles to ADMIN or EDITOR");
        }

        const updatedUser = await context.userService.updateUser(
          updateUserDto,
          requesterRole
        );
        context.loggingService.logInfo(
          `User   updated successfully: ${JSON.stringify(updatedUser)}`
        );
        return updatedUser;
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : "Unknown error occurred";
        context.loggingService.logError(
          `Error in updateUser   resolver: ${errorMessage}`
        );
        throw new Error("Could not update user: " + errorMessage);
      }
    },
  },
};
