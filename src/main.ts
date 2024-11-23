import "dotenv/config"; // Load environment variables from .env file
import { createServer } from "node:http";
import { createSchema, createYoga } from "graphql-yoga";
import { UserResolver } from "./resolvers/UserResolver"; // Adjust the path as necessary
import { UserService } from "./services/UserService"; // Adjust the path as necessary
import pg from "pg"; // Import the pg module
import { drizzle } from "drizzle-orm/node-postgres"; // Adjust this import based on your database setup
import { LoggingService } from "./utils/logger"; // Import the LoggingService

// Create a database connection pool
const { Pool } = pg; // Destructure Pool from pg
const pool = new Pool({
  connectionString:
    process.env.DATABASE_URL ||
    "postgres://postgres:mypassword@localhost:5432/test", // Use your connection string or fallback
  // Other pool options can go here
});

// Create your database instance
const db = drizzle(pool);

// Create an instance of LoggingService
const loggingService = new LoggingService(); // Adjust the instantiation as per your LoggingService implementation

// Create an instance of UserService with the database instance
const userService = new UserService(db, loggingService);

// Define your GraphQL schema and resolvers
const schema = createSchema({
  typeDefs: /* GraphQL */ `
    type User {
      discordID: String!
      tagNumber: Int
      name: String
      role: UserRole
    }

    enum UserRole {
      ADMIN
      USER
      RATTLER
    }

    input UserInput {
      discordID: String!
      tagNumber: Int
      name: String
      role: UserRole
    }

    type Query {
      getUser(discordID: String, tagNumber: Int): User
    }

    type Mutation {
      createUser(input: UserInput): User
      updateUser(input: UserInput, requesterRole: UserRole): User
    }
  `,
  resolvers: {
    Query: UserResolver.Query,
    Mutation: UserResolver.Mutation,
  },
});

// Create a Yoga server with optional configurations
const yoga = createYoga({
  schema,
  context: () => ({
    userService, // Pass the userService instance to the context
    loggingService, // Pass the loggingService instance to the context
  }),
  graphiql: true, // Enable GraphiQL for interactive API exploration
});

// Create an HTTP server and attach the Yoga server
const server = createServer(yoga);

// Start the server and listen on port 4000
const PORT = process.env.PORT || 4000; // Use environment variable or default to 4000
server.listen(PORT, () => {
  console.info(`Server is running on http://localhost:${PORT}/graphql`);
});
