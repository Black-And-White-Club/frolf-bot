// src/modules/user/user.module.ts
import { Module } from "@nestjs/common";
import { UserService } from "./user.service";
import { UserResolver } from "./user.resolver";
import { DatabaseModule } from "../../db/database.module";
import * as userSchema from "./user.model"; // Import the User schema

@Module({
  imports: [DatabaseModule.forFeature(userSchema, "USER_DATABASE_CONNECTION")], // Use forFeature
  providers: [UserResolver, UserService],
  exports: [UserService],
})
export class UserModule {}
