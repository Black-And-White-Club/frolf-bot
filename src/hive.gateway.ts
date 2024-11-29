import { readFileSync } from "fs";
import { createServer } from "http";
import { createSchema, createYoga } from "graphql-yoga";

const schema = createSchema({
  typeDefs: readFileSync("./supergraph.graphql", "utf-8"),
});

const yoga = createYoga({ schema });
const server = createServer(yoga);

server.listen(4000, () => {
  console.log("Hive Gateway running on http://localhost:4000/graphql");
});
