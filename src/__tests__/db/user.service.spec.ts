// src/__tests__/db/user.service.spec.ts
import { expect, describe, it, beforeAll, afterAll } from "vitest";
import { UserService } from "../../services/UserService";
import { User } from "../../db/models/User";
import { drizzle } from "drizzle-orm/node-postgres";
import { eq } from "drizzle-orm";
import {
  PostgreSqlContainer,
  StartedPostgreSqlContainer,
} from "@testcontainers/postgresql";
import { Client } from "pg";
import { Test, TestingModule } from "@nestjs/testing";
import { UserRole } from "../../types.generated"; // Adjust the import based on your project structure

describe("User Service", () => {
  let service: UserService;
  let db: ReturnType<typeof drizzle>;
  let client: Client;
  let postgresContainer: StartedPostgreSqlContainer;
  let module: TestingModule;

  beforeAll(async () => {
    // Disable the reaper if needed
    process.env.TESTCONTAINERS_REAPER_ENABLED = "false";

    // Start the PostgreSQL container
    postgresContainer = await new PostgreSqlContainer(
      "postgres:alpine"
    ).start();
    console.log("PostgreSQL container started with settings:", {
      host: postgresContainer.getHost(),
      port: postgresContainer.getPort(),
    });

    // Set up the PostgreSQL client with container details
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

    // Initialize the NestJS testing module
    module = await Test.createTestingModule({
      providers: [
        UserService,
        { provide: "DATABASE_CONNECTION", useValue: db },
      ],
    }).compile();

    service = module.get<UserService>(UserService);

    // Create the required user table if it doesn't exist
    await db.execute(
      `CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR,
        "discordID" VARCHAR UNIQUE,
        "tagNumber" INT,
        role VARCHAR
      )`
    );
  });

  afterAll(async () => {
    // Cleanup
    await client.end();
    await postgresContainer.stop();
  });

  it("should create a user successfully", async () => {
    const user = {
      name: "Test User1",
      discordID: "test-user-id1",
      tagNumber: 1,
    };

    const createdUser = await service.createUser(user);

    expect(createdUser).toEqual({
      __typename: "User",
      discordID: user.discordID,
      name: user.name,
      tagNumber: user.tagNumber,
      role: "RATTLER", // Default role
    });

    const dbUser = await db
      .select()
      .from(User)
      .where(eq(User.discordID, user.discordID))
      .execute();

    expect(dbUser[0]).toEqual({
      ...user,
      role: "RATTLER",
    });
  });

  it("should throw an error when creating a user with an existing Discord ID", async () => {
    const user = {
      name: "Test User",
      discordID: "test-user-id",
      tagNumber: 1,
    };

    await service.createUser(user);

    await expect(service.createUser(user)).rejects.toThrow(
      "User with this Discord ID already exists"
    );
  });

  it("should retrieve a user by Discord ID", async () => {
    const user = {
      name: "Test User3",
      discordID: "test-user-id3",
      tagNumber: 3,
    };

    await service.createUser(user);

    const retrievedUser = await service.getUserByDiscordID(user.discordID);

    expect(retrievedUser).toEqual({
      __typename: "User",
      discordID: user.discordID,
      name: user.name,
      tagNumber: user.tagNumber,
      role: "RATTLER",
    });
  });

  it("should return null when retrieving a non-existent user by Discord ID", async () => {
    const retrievedUser = await service.getUserByDiscordID("non-existent-id");
    expect(retrievedUser).toBeNull();
  });

  it("should update a user's details successfully", async () => {
    const user = {
      name: "Test User4",
      discordID: "test-user-id4",
      tagNumber: 4,
    };

    await service.createUser(user);

    const updatedUser = await service.updateUser(
      { discordID: user.discordID, name: "Updated User", tagNumber: 456 },
      UserRole.Admin // Assuming the requester has ADMIN role
    );

    expect(updatedUser).toEqual({
      __typename: "User",
      discordID: user.discordID,
      name: "Updated User",
      tagNumber: 456,
      role: "RATTLER", // Role remains unchanged
    });

    const dbUser = await db
      .select()
      .from(User)
      .where(eq(User.discordID, user.discordID))
      .execute();

    expect(dbUser[0]).toEqual({
      name: "Updated User",
      discordID: user.discordID,
      tagNumber: 456,
      role: "RATTLER",
    });
  });

  it("should throw an error when updating a user that does not exist", async () => {
    await expect(
      service.updateUser(
        { discordID: "non-existent-id", name: "Updated User" },
        UserRole.Admin
      )
    ).rejects.toThrow("User not found");
  });

  it("should throw an error when a non-admin tries to change a user's role", async () => {
    const user = {
      name: "Test User5",
      discordID: "test-user-id5",
      tagNumber: 5,
    };

    await service.createUser(user);

    await expect(
      service.updateUser(
        { discordID: user.discordID, role: UserRole.Admin },
        UserRole.Rattler // Non-admin role
      )
    ).rejects.toThrow("Only ADMIN can change roles to ADMIN or EDITOR");
  });

  it("should retrieve a user by tag number", async () => {
    const user = {
      name: "Test User6",
      discordID: "test-user-id6",
      tagNumber: 6,
    };

    await service.createUser(user);

    const retrievedUser = await service.getUserByTagNumber(user.tagNumber);

    expect(retrievedUser).toEqual({
      __typename: "User",
      discordID: user.discordID,
      name: user.name,
      tagNumber: user.tagNumber,
      role: "RATTLER",
    });
  });

  it("should return null when retrieving a non-existent user by tag number", async () => {
    const retrievedUser = await service.getUserByTagNumber(999);
    expect(retrievedUser).toBeNull();
  });
});
