import { Resolver, Query, Mutation, Args } from "@nestjs/graphql";
import { UserService } from "../services/UserService"; // Ensure this is the correct import
import {
  User as UserType,
  UserInput,
  UpdateUserInput,
  UserRole,
} from "../types.generated"; // Import generated types
import { CreateUserDto } from "../dto/create-user.dto"; // Import CreateUser  Dto
import { UpdateUserDto } from "../dto/update-user.dto"; // Import UpdateUser  Dto
import { plainToClass } from "class-transformer"; // Importing for transformation
import { validate } from "class-validator"; // Importing for validation

@Resolver() // Specify UserType as the resolver's type
export class UserResolver {
  constructor(private readonly userService: UserService) {} // Inject UserService as an instance

  @Mutation() // Specify UserType as the return type
  async createUser(@Args("input") input: UserInput): Promise<UserType> {
    const createUserDto = plainToClass(CreateUserDto, input);
    const errors = await validate(createUserDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!"); // You can customize this error handling
    }
    return await this.userService.createUser(createUserDto);
  }

  @Mutation() // Specify UserType as the return type
  async updateUser(
    @Args("input") input: UpdateUserInput,
    @Args("requesterRole") requesterRole: UserRole
  ): Promise<UserType> {
    const updateUserDto = plainToClass(UpdateUserDto, input);
    const errors = await validate(updateUserDto);
    if (errors.length > 0) {
      throw new Error("Validation failed!"); // You can customize this error handling
    }

    // Check if the requesterRole is Admin
    if (requesterRole !== UserRole.Admin) {
      throw new Error("Only Admin can update user roles to Editor or Admin."); // Custom error message
    }

    // Check if the new role is Editor or Admin
    if (
      updateUserDto.role === UserRole.Editor ||
      updateUserDto.role === UserRole.Admin
    ) {
      // You can also add additional checks here if needed
      return await this.userService.updateUser(updateUserDto, requesterRole);
    } else {
      // Proceed with the update if the role is not Editor or Admin
      return await this.userService.updateUser(updateUserDto, requesterRole);
    }
  }

  @Query() // Specify UserType as the return type and allow null
  async getUser(
    @Args("discordID") discordID: string
  ): Promise<UserType | null> {
    return await this.userService.getUserByDiscordID(discordID);
  }
}
