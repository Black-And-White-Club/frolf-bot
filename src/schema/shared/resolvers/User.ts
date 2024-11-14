import type { UserResolvers } from "../../types.generated";

export const User: UserResolvers = {
  // Resolver for the 'id' field
  id: (user) => user.id,

  // Resolver for the 'name' field
  name: (user) => user.name,

  // Resolver for the 'discordID' field
  discordID: (user) => user.discordID,

  // Resolver for the optional 'tagNumber' field
  tagNumber: (user) => user.tagNumber,

  // Resolver for the 'rounds' field (assuming you have a Round type defined elsewhere)
  rounds: (user, _, { dataSources }) => {
    // Assuming you have a data source that fetches rounds for a user
    return dataSources.roundAPI.getRoundsByUserId(user.id);
  },

  // Resolver for the 'role' field
  role: (user) => user.role,

  // Resolver for the 'editLog' field
  editLog: (user) => user.editLog,
};
