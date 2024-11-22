import { plainToClass } from "class-transformer";
import { validate } from "class-validator";
import { UserInput, UserRole } from "../types.generated";
import { UserService } from "../services/UserService";
import { CreateUserDto } from "../dto/create-user.dto";
import { UpdateUserDto } from "../dto/update-user.dto";

export const UserResolver = {
  Query: {
    async getUser(
      _: any,
      args: { discordID?: string; tagNumber?: number },
      context: { userService: UserService }
    ) {
      if (args.discordID) {
        const user = await context.userService.getUserByDiscordID(
          args.discordID
        );
        if (!user) {
          throw new Error("User not found by discordID");
        }
        return user;
      } else if (args.tagNumber) {
        const user = await context.userService.getUserByTagNumber(
          args.tagNumber
        );
        if (!user) {
          throw new Error("User not found by Tag");
        }
        return user;
      } else {
        throw new Error("Please provide either discordID or tagNumber");
      }
    },
  },
  Mutation: {
    async createUser(
      _: any,
      args: { input: UserInput },
      context: { userService: UserService }
    ) {
      const createUserDto = plainToClass(CreateUserDto, args.input);
      const errors = await validate(createUserDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }
      return await context.userService.createUser(createUserDto);
    },

    async updateUser(
      _: any,
      args: { input: UpdateUserDto; requesterRole: UserRole },
      context: { userService: UserService }
    ) {
      const { input, requesterRole } = args;

      // Fetch the current user based on the provided discordID (if it exists)
      const currentUser = await context.userService.getUserByDiscordID(
        input.discordID
      );

      if (!currentUser) {
        throw new Error("User not found");
      }

      // Ensure that discordID and role are populated with defaults if not provided
      const updateUserDto: UpdateUserDto = {
        discordID: input.discordID || currentUser.discordID, // Ensuring discordID is not undefined
        role: input.role || currentUser.role, // Ensuring role is not undefined
        name: input.name || currentUser.name, // If name is provided, update it
        tagNumber:
          input.tagNumber === undefined
            ? currentUser.tagNumber
            : input.tagNumber, // Handle undefined or null tagNumber
      };

      // Validate the DTO
      const errors = await validate(updateUserDto);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      // Perform the update
      return await context.userService.updateUser(updateUserDto, requesterRole);
    },
  },
};
