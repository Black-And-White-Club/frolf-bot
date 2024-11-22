import { YogaDriver, YogaDriverConfig } from "@graphql-yoga/nestjs";
import { Module } from "@nestjs/common";
import { GraphQLModule } from "@nestjs/graphql";

@Module({
  imports: [
    GraphQLModule.forRoot<YogaDriverConfig>({
      driver: YogaDriver,
    }),
  ],
})
export class AppModule {}
