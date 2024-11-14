import type { CodegenConfig } from "@graphql-codegen/cli";
import { defineConfig } from "@eddeee888/gcg-typescript-resolver-files";

const config: CodegenConfig = {
  schema: "**/schema.graphql",
  generates: {
    "src/schema": defineConfig({
      // The following config is designed to work with GraphQL Yoga's File uploads feature
      // https://the-guild.dev/graphql/yoga-server/docs/features/file-uploads
      scalarsOverrides: {
        File: { type: "File" },
      },
      resolverGeneration: {
        query: "*",
        mutation: "*",
        subscription: "*",
        scalar: "!*.File",
        object: "*",
        union: "",
        interface: "",
      },
    }),
  },
};
export default config;
