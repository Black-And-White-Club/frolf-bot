import { Module } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";
import { YogaDriver, YogaDriverConfig } from "@graphql-yoga/nestjs";
import { UserService } from "./services/UserService";
import { UserResolver } from "./resolvers/UserResolver";

@Module({
  imports: [
    GraphQLModule.forRoot<YogaDriverConfig>({
      driver: YogaDriver,
      autoSchemaFile: true,
    }),
  ],
  providers: [UserService, UserResolver],
})
export class AppModule {}
