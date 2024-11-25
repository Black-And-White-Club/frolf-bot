// app.module.ts
import { Module } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import { YogaDriver } from "@graphql-yoga/nestjs";
import { join } from "path";
import { UserModule } from "./modules/user/user.module";
import { UserResolver } from "./modules/user/user.resolver"; // Import UserResolver
import { ValidationPipe } from "@nestjs/common";

@Module({
  imports: [
    // ... your GraphQLModule configuration
    UserModule,
  ],
  providers: [
    UserResolver, // Provide UserResolver here
  ],
})
export class AppModule {}
