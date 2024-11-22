import { UserResolver } from "./UserResolver";

export const resolvers = {
  Query: {
    ...UserResolver.Query,
  },
  Mutation: {
    ...UserResolver.Mutation,
  },
};
