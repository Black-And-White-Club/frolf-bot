// src/modules/user/user.module.ts
import { Module } from "@nestjs/common";
import { UserService } from "./user.service";
import { UserResolver } from "./user.resolver";
import { DatabaseModule } from "../../db/database.module";

@Module({
  imports: [DatabaseModule],
  providers: [UserResolver, UserService],
  exports: [UserService],
})
export class UserModule {}
