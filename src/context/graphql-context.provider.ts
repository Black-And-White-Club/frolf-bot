// src/context/graphql-context.provider.ts
import { Provider } from "@nestjs/common";
import { REQUEST } from "@nestjs/core";
import { v4 as uuidv4 } from "uuid";

export class GraphQLContext {
  reqId: string | undefined;
  // ... other context data you might need in the future
}

export const GraphQLContextProvider: Provider = {
  provide: GraphQLContext,
  useFactory: ({ req }) => {
    // Access req directly from the context object
    const context = new GraphQLContext();
    context.reqId = uuidv4();
    // ... (any other context initialization logic) ...
    return context;
  },
  inject: [REQUEST], // Inject the REQUEST token
};
