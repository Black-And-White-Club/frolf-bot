// src/modules/user/user.module.ts
import { Module } from "@nestjs/common";
import { UserService } from "./user.service";
import { UserResolver } from "./user.resolver";
import { DatabaseModule } from "src/db/database.module";
import * as userSchema from "./user.model";

@Module({
  imports: [DatabaseModule.forFeature(userSchema, "USER_DATABASE_CONNECTION")],
  providers: [UserResolver, UserService],
  exports: [UserService],
})
export class UserModule {}
