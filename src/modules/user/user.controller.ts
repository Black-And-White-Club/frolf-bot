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
@UseGuards(AuthGuard)
export class UserController {
  constructor(private readonly userService: UserService) {}

  @Post()
  async createUser(@Body() createUserDto: CreateUserDto): Promise<User> {
    const user = new User();
    user.name = createUserDto.name;
    user.discordID = createUserDto.discordID;
    user.role = createUserDto.role;
    if (createUserDto.tagNumber !== undefined) {
      user.tagNumber = createUserDto.tagNumber;
    }
    return this.userService.createUser(user);
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
