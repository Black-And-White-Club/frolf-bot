import { expect, describe, it, beforeAll, afterAll, beforeEach } from "vitest";
import { LeaderboardService } from "../../services/LeaderboardService";
import { drizzle } from "drizzle-orm/node-postgres";
import {
  PostgreSqlContainer,
  StartedPostgreSqlContainer,
} from "@testcontainers/postgresql";
import { Client } from "pg";
import { Test, TestingModule } from "@nestjs/testing";

describe("Leaderboard Service", () => {
  let service: LeaderboardService;
  let db: ReturnType<typeof drizzle>;
  let client: Client;
  let postgresContainer: StartedPostgreSqlContainer;

  beforeAll(async () => {
    process.env.TESTCONTAINERS_REAPER_ENABLED = "false";

    postgresContainer = await new PostgreSqlContainer(
      "postgres:alpine"
    ).start();

    client = new Client({
      host: postgresContainer.getHost(),
      port: postgresContainer.getPort(),
      database: postgresContainer.getDatabase(),
      user: postgresContainer.getUsername(),
      password: postgresContainer.getPassword(),
    });

    await client.connect();
    db = drizzle({ client });

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        LeaderboardService,
        { provide: "DATABASE_CONNECTION", useValue: db },
      ],
    }).compile();

    service = module.get<LeaderboardService>(LeaderboardService);

    // Create the leaderboard table
    await db.execute(`
      CREATE TABLE IF NOT EXISTS leaderboard (
        "discordID" VARCHAR NOT NULL,
        "tagNumber" INT NOT NULL,
        "lastPlayed" TIMESTAMP,
        "durationHeld" INT DEFAULT 0,
        "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
        "updated_at" TIMESTAMP DEFAULT NOW() NOT NULL,
        "deleted_at" TIMESTAMP
      )
    `);
  });

  afterAll(async () => {
    await client.end();
    await postgresContainer.stop();
  });

  beforeEach(async () => {
    // Reset the leaderboard table between tests
    await db.execute(`TRUNCATE leaderboard`);
  });

  it("should link a tag successfully for user1", async () => {
    const discordID = "user1";
    const tagNumber = 100;

    const linkedTag = await service.linkTag(discordID, tagNumber);

    expect(linkedTag).toEqual(
      expect.objectContaining({
        discordID,
        tagNumber,
      })
    );

    const dbEntry = await service.getUserTag(discordID);
    expect(dbEntry).toEqual(
      expect.objectContaining({
        discordID,
        tagNumber,
      })
    );
  });

  it("should process scores successfully after linking", async () => {
    const users = [
      { discordID: "user1", tagNumber: 100 },
      { discordID: "user2", tagNumber: 50 }, // Lower score (better)
      { discordID: "user3", tagNumber: 150 },
      { discordID: "user4", tagNumber: 25 }, // Best score
    ];

    for (const user of users) {
      await service.linkTag(user.discordID, user.tagNumber);
    }

    const leaderboard = await service.getLeaderboard(1, 10);

    expect(leaderboard.users).toHaveLength(4);
    expect(leaderboard.users[0].discordID).toBe("user4"); // Best score first
    expect(leaderboard.users[1].discordID).toBe("user2");
    expect(leaderboard.users[2].discordID).toBe("user1");
    expect(leaderboard.users[3].discordID).toBe("user3");
  });

  it("should throw an error when linking a tag that is already taken", async () => {
    const discordID = "user5";
    const tagNumber = 500;

    await service.linkTag(discordID, tagNumber);

    await expect(service.linkTag(discordID, tagNumber)).rejects.toThrow(
      `Tag number ${tagNumber} is already taken.`
    );
  });

  it("should handle invalid discordID gracefully", async () => {
    await expect(service.linkTag("", 600)).rejects.toThrow(
      "discordID cannot be empty."
    );
  });

  it("should handle multiple users correctly", async () => {
    const users = [
      { discordID: "user6", tagNumber: 120 },
      { discordID: "user7", tagNumber: 90 }, // Better score
      { discordID: "user8", tagNumber: 200 },
      { discordID: "user9", tagNumber: 70 }, // Best score
    ];

    for (const user of users) {
      await service.linkTag(user.discordID, user.tagNumber);
    }

    const leaderboard = await service.getLeaderboard(1, 10);

    expect(leaderboard.users).toHaveLength(4);
    expect(leaderboard.users[0].discordID).toBe("user9"); // Best score first
    expect(leaderboard.users[1].discordID).toBe("user7");
    expect(leaderboard.users[2].discordID).toBe("user6");
    expect(leaderboard.users[3].discordID).toBe("user8");
  });
});
