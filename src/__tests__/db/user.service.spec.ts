import { expect, describe, it, beforeAll, afterAll } from "vitest";
import { UserService } from "../../modules/user/user.service";
import { UserRole } from "../../enums/user-role.enum";
import { drizzle } from "drizzle-orm/node-postgres";
import {
  PostgreSqlContainer,
  StartedPostgreSqlContainer,
} from "@testcontainers/postgresql";
import { Client } from "pg";
import { Test, TestingModule } from "@nestjs/testing";
import { users as UserModel } from "../../modules/user/user.model"; // Ensure this import is correct

describe("UserService with TestContainers", () => {
  let userService: UserService;
  let db: ReturnType<typeof drizzle>;
  let client: Client;
  let postgresContainer: StartedPostgreSqlContainer;
  let module: TestingModule;

  beforeAll(async () => {
    process.env.TESTCONTAINERS_REAPER_ENABLED = "false";

    // Start PostgreSQL container
    postgresContainer = await new PostgreSqlContainer(
      "postgres:alpine"
    ).start();

    // Set up PostgreSQL client with container details
    client = new Client({
      host: postgresContainer.getHost(),
      port: postgresContainer.getPort(),
      database: postgresContainer.getDatabase(),
      user: postgresContainer.getUsername(),
      password: postgresContainer.getPassword(),
    });

    // Connect to the database
    await client.connect();

    // Initialize Drizzle ORM with the client
    db = drizzle({ client });

    // Initialize the NestJS testing module with the real services
    module = await Test.createTestingModule({
      providers: [
        UserService,
        { provide: "DATABASE_CONNECTION", useValue: db },
      ],
    }).compile();

    userService = module.get<UserService>(UserService);

    // Create the users table if it doesn't exist
    try {
      await db.execute(`
        CREATE TABLE IF NOT EXISTS users (
          id SERIAL PRIMARY KEY,
          "discordID" VARCHAR(255) NOT NULL UNIQUE,
          name VARCHAR(255) NOT NULL,
          "tagNumber" INT UNIQUE,
          role VARCHAR(50) NOT NULL DEFAULT 'RATTLER',
          "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
          "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
          "deleted_at" TIMESTAMP DEFAULT NULL
        )
      `);
    } catch (error) {
      console.error("Error creating table:", error);
      throw error; // Fail the test suite setup
    }
  });

  describe("getUserByDiscordID", () => {
    it("should fetch a user by Discord ID", async () => {
      // Arrange: Use the service to create a user
      await userService.createUser({
        discordID: "123456",
        name: "TestUser",
        tagNumber: 42,
        role: UserRole.ADMIN,
      });

      // Act: Call the service method to fetch the user
      const user = await userService.getUserByDiscordID("123456");

      // Assert: Verify the user data matches
      expect(user).toMatchObject({
        discordID: "123456",
        name: "TestUser",
        tagNumber: 42,
        role: UserRole.ADMIN,
        createdAt: expect.any(String),
        updatedAt: expect.any(String),
      });
    });

    it("should return null if no user is found", async () => {
      // Act: Call the service method with a non-existent Discord ID
      const user = await userService.getUserByDiscordID("non-existent-id");

      // Assert: Verify null is returned
      expect(user).toBeNull();
    });
  });

  describe("createUser", () => {
    it("should create a new user", async () => {
      // Arrange: Prepare user data
      const userData = {
        discordID: "987654",
        name: "NewUser",
        tagNumber: 99,
        role: UserRole.EDITOR,
      };

      // Act: Call the service method to create a user
      const createdUser = await userService.createUser(userData);

      // Assert: Verify the created user response
      expect(createdUser).toMatchObject({
        discordID: "987654",
        name: "NewUser",
        tagNumber: 99,
        role: UserRole.EDITOR,
      });
    });
  });

  describe("updateUser", () => {
    it("should update an existing user", async () => {
      // Arrange: Use the service to create a user
      await userService.createUser({
        discordID: "111111",
        name: "OldName",
        tagNumber: 123,
        role: UserRole.RATTLER,
      });

      // Act: Call the service method to update the user
      const updatedUser = await userService.updateUser(
        { discordID: "111111", name: "UpdatedName", role: UserRole.EDITOR },
        UserRole.ADMIN
      );

      // Assert: Verify the updated user response
      expect(updatedUser).toMatchObject({
        discordID: "111111",
        name: "UpdatedName",
        tagNumber: 123,
        role: UserRole.EDITOR,
      });
    });

    it("should throw an error if the user does not exist", async () => {
      // Act & Assert: Attempt to update a non-existent user
      await expect(
        userService.updateUser(
          { discordID: "non-existent-id", name: "NewName" },
          UserRole.ADMIN
        )
      ).rejects.toThrow("User not found");
    });
  });
});
