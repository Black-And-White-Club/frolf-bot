import { expect, describe, it, beforeEach, afterEach } from "vitest";
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

describe.only("User Service", () => {
  let service: UserService;
  let db: ReturnType<typeof drizzle>;
  let client: Client;
  let postgresContainer: StartedPostgreSqlContainer;
  let module: TestingModule;

  beforeEach(async () => {
    // Disable the reaper if needed
    process.env.TESTCONTAINERS_REAPER_ENABLED = "false";

    // Start the PostgreSQL container
    postgresContainer = await new PostgreSqlContainer().start();
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
      "CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name VARCHAR, discordID VARCHAR, tagNumber INT, role VARCHAR)"
    );
  });

  afterEach(async () => {
    // Cleanup
    await client.end();
    await postgresContainer.stop();
  });

  it("should create a user successfully", async () => {
    const user = {
      name: "Test User ",
      discordID: "test-user-id",
      tagNumber: 123,
      role: "Rattler",
    };

    const createdUser = await service.createUser(user);

    expect(createdUser).toEqual({
      __typename: "User ",
      ...user,
    });

    const dbUser = await db
      .select()
      .from(User)
      .where(eq(User.discordID, user.discordID))
      .execute();

    expect(dbUser[0]).toEqual(user);
  });
});
