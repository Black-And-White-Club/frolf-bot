import { expect, describe, it, beforeAll, afterAll } from "vitest";
import { ScoreService } from "../../services/ScoreService";
import { Score } from "../../db/models/Score"; // Ensure this import is correct
import { drizzle } from "drizzle-orm/node-postgres";
import { eq, and } from "drizzle-orm";
import {
  PostgreSqlContainer,
  StartedPostgreSqlContainer,
} from "@testcontainers/postgresql";
import { Client } from "pg";
import { Test, TestingModule } from "@nestjs/testing";

describe("Score Service", () => {
  let service: ScoreService;
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
        ScoreService,
        { provide: "DATABASE_CONNECTION", useValue: db },
      ],
    }).compile();

    service = module.get<ScoreService>(ScoreService);

    // Create the required score table if it doesn't exist
    await db.execute(
      `CREATE TABLE IF NOT EXISTS scores (
        id SERIAL PRIMARY KEY,
        discordID VARCHAR,
        roundID VARCHAR,
        score INT,
        tagNumber INT
      )`
    );
  });

  afterAll(async () => {
    // Cleanup
    await client.end();
    await postgresContainer.stop();
  });

  it("should create a score successfully", async () => {
    const scoreInput = {
      discordID: "test-user-id1",
      roundID: "round-id1",
      score: 100,
      tagNumber: 1,
    };

    const createdScore = await service.processScores(scoreInput.roundID, [
      scoreInput,
    ]);

    expect(createdScore).toEqual([
      {
        __typename: "Score",
        discordID: scoreInput.discordID,
        score: scoreInput.score,
        tagNumber: scoreInput.tagNumber,
      },
    ]);

    const dbScore = await db
      .select()
      .from(Score)
      .where(eq(Score.discordID, scoreInput.discordID))
      .execute();

    expect(dbScore[0]).toEqual({
      ...scoreInput,
    });
  });

  it("should throw an error when creating a score for an existing Discord ID", async () => {
    const scoreInput = {
      discordID: "test-user-id2",
      roundID: "round-id2",
      score: 200,
      tagNumber: 2,
    };

    await service.processScores(scoreInput.roundID, [scoreInput]);

    await expect(
      service.processScores(scoreInput.roundID, [scoreInput])
    ).rejects.toThrow(
      "Score for this Discord ID already exists for the given round"
    );
  });

  it("should retrieve a score by Discord ID and Round ID", async () => {
    const scoreInput = {
      discordID: "test-user-id3",
      roundID: "round-id3",
      score: 300,
      tagNumber: 3,
    };

    await service.processScores(scoreInput.roundID, [scoreInput]);

    const retrievedScore = await service.getUserScore(
      scoreInput.discordID,
      scoreInput.roundID
    );

    expect(retrievedScore).toEqual({
      __typename: "Score",
      discordID: scoreInput.discordID, // Corrected this line
      score: scoreInput.score,
      tagNumber: scoreInput.tagNumber,
    });
  });

  it("should return null when retrieving a non-existent score by Discord ID and Round ID", async () => {
    const retrievedScore = await service.getUserScore(
      "non-existent-id",
      "non-existent-round"
    );
    expect(retrievedScore).toBeNull();
  });

  it("should retrieve a score by Discord ID and Round ID", async () => {
    const scoreInput = {
      discordID: "test-user-id3",
      roundID: "round-id3",
      score: 300,
      tagNumber: 3,
    };

    await service.processScores(scoreInput.roundID, [scoreInput]);

    const retrievedScore = await service.getUserScore(
      scoreInput.discordID,
      scoreInput.roundID
    );

    expect(retrievedScore).toEqual({
      __typename: "Score",
      discordID: scoreInput.discordID,
      score: scoreInput.score,
      tagNumber: scoreInput.tagNumber,
    });

    const dbScore = await db
      .select()
      .from(Score)
      .where(
        and(
          eq(Score.discordID, scoreInput.discordID),
          eq(Score.roundID, scoreInput.roundID)
        )
      )
      .execute();

    expect(dbScore[0]).toEqual({
      discordID: scoreInput.discordID,
      roundID: scoreInput.roundID,
      score: scoreInput.score,
      tagNumber: scoreInput.tagNumber,
    });
  });

  it("should throw an error when updating a score that does not exist", async () => {
    await expect(
      service.updateScore("non-existent-id", "non-existent-round", 500)
    ).rejects.toThrow("Score not found");
  });
});
