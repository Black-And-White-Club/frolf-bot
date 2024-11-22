import { createYoga } from "graphql-yoga";
import { makeExecutableSchema } from "@graphql-tools/schema";
import { typeDefs } from "./schema";
import { resolvers } from "./resolvers";
import { UserService } from "./services/UserService";
import { drizzle } from "drizzle-orm/node-postgres"; // Import Drizzle ORM
import { Client } from "pg"; // Import PostgreSQL client
import "reflect-metadata";

// Create a PostgreSQL client
const client = new Client({
  host: "localhost", // Replace with your actual DB host
  port: 5432, // Replace with your actual DB port
  database: "your_database_name", // Replace with your actual DB name
  user: "your_username", // Replace with your actual DB user
  password: "your_password", // Replace with your actual DB password
});

// Connect to the database
client.connect();

// Initialize Drizzle ORM with the client
const db = drizzle(client);

// Create the UserService instance with the db
const userService = new UserService(db);

// Create the GraphQL schema
const schema = makeExecutableSchema({
  typeDefs,
  resolvers,
});

// Create the Yoga server
const yoga = createYoga({
  schema,
  context: () => ({
    userService, // Provide the userService instance in the context
  }),
});

// Export the Yoga server
export default yoga;
