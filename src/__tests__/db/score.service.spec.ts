import { expect, describe, it, beforeAll, afterAll } from "vitest";
import { ScoreService } from "../../modules/score/score.service";
import { scores } from "../../modules/score/score.model";
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
    await db.execute(`
      CREATE TABLE IF NOT EXISTS scores (
        id SERIAL PRIMARY KEY,
        "discordID" VARCHAR,
        "roundID" VARCHAR,
        score INT,
        "tagNumber" INT,
        "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
        "updated_at" TIMESTAMP DEFAULT NOW() NOT NULL,
        "deleted_at" TIMESTAMP
      )
    `);
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
      score: -5,
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
      .from(scores)
      .where(eq(scores.discordID, scoreInput.discordID))
      .execute();

    expect(dbScore[0]).toEqual(
      expect.objectContaining({
        ...scoreInput,
      })
    );
  });

  it("should throw an error when creating a score for an existing Discord ID", async () => {
    const scoreInput = {
      discordID: "test-user-id2",
      roundID: "round-id2",
      score: +13,
      tagNumber: 2,
    };

    // First creation should succeed
    await service.processScores(scoreInput.roundID, [scoreInput]);

    // Second creation should throw an error
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
      score: +2,
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
  });

  it("should return null when retrieving a non-existent score by Discord ID and Round ID", async () => {
    const retrievedScore = await service.getUserScore(
      "non-existent-id",
      "non-existent-round"
    );
    expect(retrievedScore).toBeNull();
  });

  it("should throw an error when updating a score that does not exist", async () => {
    await expect(
      service.updateScore("non-existent-id", "non-existent-round", -5)
    ).rejects.toThrow("Score not found");
  });

  it("should update an existing score successfully", async () => {
    const scoreInput = {
      discordID: "test-user-id5",
      roundID: "round-id5",
      score: +10,
      tagNumber: 5,
    };

    await service.processScores(scoreInput.roundID, [scoreInput]);

    const updatedScore = await service.updateScore(
      scoreInput.discordID,
      scoreInput.roundID,
      +15,
      6
    );

    expect(updatedScore).toEqual({
      __typename: "Score",
      discordID: scoreInput.discordID,
      score: +15,
      tagNumber: 6,
    });

    const dbScore = await db
      .select()
      .from(scores)
      .where(
        and(
          eq(scores.discordID, scoreInput.discordID),
          eq(scores.roundID, scoreInput.roundID)
        )
      )
      .execute();

    expect(dbScore[0]).toEqual(
      expect.objectContaining({
        discordID: scoreInput.discordID,
        roundID: scoreInput.roundID,
        score: +15,
        tagNumber: 6,
      })
    );
  });

  it("should retrieve all scores for a specific round", async () => {
    const roundID = "round-id6";
    const scoresInput = [
      { discordID: "user1", score: -5, tagNumber: 1 },
      { discordID: "user2", score: -10, tagNumber: 2 },
    ];

    await service.processScores(roundID, scoresInput);

    const retrievedScores = await service.getScoresForRound(roundID);

    expect(retrievedScores).toEqual(
      scoresInput.map((score) => ({
        __typename: "Score",
        discordID: score.discordID,
        score: score.score,
        tagNumber: score.tagNumber,
      }))
    );
  });

  it("should return an empty array when there are no scores for a specific round", async () => {
    const retrievedScores = await service.getScoresForRound(
      "non-existent-round"
    );
    expect(retrievedScores).toEqual([]);
  });
});
