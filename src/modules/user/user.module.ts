import { Module } from "@nestjs/common";
import { UserService } from "./user.service";
import { UserResolver } from "./user.resolver";
import { databaseProvider } from "../../db";

@Module({
  providers: [UserService, UserResolver],
  exports: [UserService],
})
export class UserModule {}
