// src/users/users.controller.ts
import {
  Controller,
  UseGuards,
  Post,
  Body,
  Get,
  Param,
  Put,
  NotFoundException,
  ForbiddenException,
} from "@nestjs/common";
import { UserService } from "./user.service";
import { CreateUserDto } from "src/dto/user/create-user.dto";
import { UpdateUserDto } from "src/dto/user/update-user.dto";
import { UserRole } from "src/enums";
import { AuthGuard } from "src/middleware/auth.guard";
import { User } from "src/modules/user/user.entity";

@Controller("users")
// @UseGuards(AuthGuard)
export class UserController {
  constructor(private readonly userService: UserService) {
    console.log("UserController constructor called", this.userService);
  }

  @Post()
  async createUser(@Body() createUserDto: CreateUserDto): Promise<User> {
    console.log("Received createUserDto:", createUserDto);

    const user = new User();
    user.name = createUserDto.name;
    user.discordID = createUserDto.discordID;
    user.role = createUserDto.role;
    if (createUserDto.tagNumber !== undefined) {
      user.tagNumber = createUserDto.tagNumber;
    }

    console.log("Calling userService.createUser with:", user);
    try {
      const result = await this.userService.createUser(user);
      console.log("userService.createUser result:", result);
      return result;
    } catch (error) {
      console.error("Error in createUser:", error);
      throw error;
    }
  }

  @Get(":discordID")
  async getUser(@Param("discordID") discordID: string): Promise<User | null> {
    return this.userService.getUserByDiscordID(discordID);
  }

  @Put(":discordID")
  async updateUser(
    @Param("discordID") discordID: string,
    @Body() updateUserDto: UpdateUserDto
  ): Promise<User> {
    const currentUser = await this.userService.getUserByDiscordID(discordID);
    if (!currentUser) {
      throw new NotFoundException("User not found");
    }
    if (
      updateUserDto.role &&
      (updateUserDto.role === UserRole.Admin ||
        updateUserDto.role === UserRole.Editor) &&
      currentUser.role !== UserRole.Admin
    ) {
      throw new ForbiddenException(
        "Only ADMIN can change roles to ADMIN or EDITOR"
      );
    }

    const updates = {
      ...updateUserDto,
      tagNumber:
        updateUserDto.tagNumber === null ? undefined : updateUserDto.tagNumber,
    };

    return this.userService.updateUser(discordID, updates);
  }
}
