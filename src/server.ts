import { createYoga } from "graphql-yoga";
import { makeExecutableSchema } from "@graphql-tools/schema";
import { typeDefs } from "./schema";
import { resolvers } from "./resolvers";
import { UserService } from "./services/UserService";

// Create the GraphQLSchema
const schema = makeExecutableSchema({
  typeDefs,
  resolvers,
});

// Create the Yoga server
const yoga = createYoga({
  schema, // Pass the combined schema here
  context: () => ({
    userService: new UserService(),
  }),
});

export default yoga;
