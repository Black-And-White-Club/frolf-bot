// codegen.ts
import { CodegenConfig } from "@graphql-codegen/cli";
import { defineConfig } from "@eddeee888/gcg-typescript-resolver-files";

const config: CodegenConfig = {
  schema: "./src/schema/**/*.graphql", // Path to your GraphQL schema files
  generates: {
    "./src/types.generated.ts": {
      plugins: ["typescript", "typescript-resolvers"],
      config: {
        avoidOptionals: false, // Allow optional fields to be generated as nullable
      },
    },

    "./src/": defineConfig({
      resolverGeneration: "disabled",
    }),
  },
};

export default config;
