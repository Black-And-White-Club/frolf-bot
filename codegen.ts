// codegen.ts
import { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "./src/schema/**/*.graphql", // Path to your GraphQL schema files
  generates: {
    "./src/types.generated.ts": {
      plugins: ["typescript", "typescript-resolvers"],
      config: {
        avoidOptionals: false, // Allow optional fields to be generated as nullable
      },
    },
  },
};

export default config;
