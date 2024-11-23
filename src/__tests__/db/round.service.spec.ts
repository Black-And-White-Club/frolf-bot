import { expect, describe, it, beforeAll, afterAll, afterEach } from "vitest";
import { RoundService } from "../../services/RoundService";
import { RoundModel } from "../../db/models/Round";
import { drizzle } from "drizzle-orm/node-postgres";
import { eq } from "drizzle-orm";
import {
  PostgreSqlContainer,
  StartedPostgreSqlContainer,
} from "@testcontainers/postgresql";
import { Client } from "pg";
import { Test, TestingModule } from "@nestjs/testing";
import { RoundState } from "../../types.generated";
import { ScoreService } from "../../services/ScoreService";
import { Response } from "../../types.generated"; // Assuming Response is an enum or type you define elsewhere

describe("Round Service", () => {
  let service: RoundService;
  let db: ReturnType<typeof drizzle>;
  let client: Client;
  let postgresContainer: StartedPostgreSqlContainer;
  let module: TestingModule;
  let scoreService: ScoreService;

  beforeAll(async () => {
    process.env.TESTCONTAINERS_REAPER_ENABLED = "false";

    // Start the PostgreSQL container
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

    module = await Test.createTestingModule({
      providers: [
        RoundService,
        ScoreService,
        { provide: "DATABASE_CONNECTION", useValue: db },
      ],
    }).compile();

    service = module.get<RoundService>(RoundService);
    scoreService = module.get<ScoreService>(ScoreService);

    await db.execute(`
      CREATE TABLE IF NOT EXISTS rounds (
        "roundID" SERIAL PRIMARY KEY,
        title VARCHAR NOT NULL,
        location VARCHAR NOT NULL,
        "eventType" VARCHAR,
        date DATE NOT NULL,
        time TIME NOT NULL,
        participants JSON NOT NULL,
        scores JSON NOT NULL,
        finalized BOOLEAN DEFAULT false,
        "creatorID" VARCHAR NOT NULL,
        state VARCHAR NOT NULL,
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

  it("should create a round successfully", async () => {
    const input = {
      title: "Test Round",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id", // Use creatorID instead of discordID
    };

    const createdRound = await service.scheduleRound(input);

    // Check that the round has been created with the correct data
    expect(createdRound).toHaveProperty("roundID");
    expect(createdRound.title).toBe(input.title);
    expect(createdRound.location).toBe(input.location);
    expect(createdRound.creatorID).toBe(input.creatorID); // Ensure creatorID is set correctly

    // Fetch the round from the database to verify it was inserted correctly
    const dbRound = await db
      .select()
      .from(RoundModel)
      .where(eq(RoundModel.roundID, Number(createdRound.roundID)))
      .execute();

    // Use expect.objectContaining to ignore createdAt, updatedAt, and deletedAt
    expect(dbRound[0]).toEqual(
      expect.objectContaining({
        roundID: createdRound.roundID,
        title: input.title,
        location: input.location,
        eventType: input.eventType,
        date: input.date,
        time: input.time,
        participants: [], // Change from "[]" to []
        scores: [], // Change from "[]" to []
        finalized: false,
        creatorID: input.creatorID, // Ensure creatorID is set correctly
        state: RoundState.Upcoming,
      })
    );
  });

  it("should join a round successfully", async () => {
    const input = {
      title: "Test Round Join",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    // Create the round
    const round = await service.scheduleRound(input);

    const joinInput = {
      roundID: round.roundID,
      discordID: "user-id",
      response: Response.Accept,
      tagNumber: 1,
    };

    const updatedRound = await service.joinRound(joinInput);

    // Check that the participant has been added
    expect(updatedRound.participants).toHaveLength(1);
    expect(updatedRound.participants[0].discordID).toBe(joinInput.discordID);
  });

  it("should finalize a round successfully", async () => {
    const input = {
      title: "Test Round Finalize",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    const round = await service.scheduleRound(input);

    // Simulate joining the round
    await service.joinRound({
      roundID: round.roundID,
      discordID: "user-id",
      response: Response.Accept,
      tagNumber: 1,
    });

    // Finalize the round
    const finalizedRound = await service.finalizeAndProcessScores(
      round.roundID,
      scoreService
    );

    // Check that the round is finalized
    expect(finalizedRound.state).toBe(RoundState.Finalized);
    expect(finalizedRound.finalized).toBe(true);
  });

  it("should throw an error when trying to join a finalized round", async () => {
    const input = {
      title: "Test Round Finalize Error",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    const round = await service.scheduleRound(input);
    await service.finalizeAndProcessScores(round.roundID, scoreService);

    const joinInput = {
      roundID: round.roundID,
      discordID: "user-id",
      response: Response.Accept,
      tagNumber: 1,
    };

    await expect(service.joinRound(joinInput)).rejects.toThrow(
      "You can only join rounds that are upcoming"
    );
  });

  it("should throw an error when trying to finalize an already finalized round", async () => {
    const input = {
      title: "Test Round Already Finalized",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    const round = await service.scheduleRound(input);
    await service.finalizeAndProcessScores(round.roundID, scoreService);

    await expect(
      service.finalizeAndProcessScores(round.roundID, scoreService)
    ).rejects.toThrow("Round has already been finalized");
  });

  it("should delete a round successfully", async () => {
    const input = {
      title: "Test Round Delete",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    const round = await service.scheduleRound(input);
    const deleteResult = await service.deleteRound(
      round.roundID,
      input.creatorID
    );

    expect(deleteResult).toBe(true);

    // Verify that the round has been deleted
    const deletedRound = await service.getRound(round.roundID);
    expect(deletedRound).toBeNull();
  });

  it("should throw an error when trying to delete a round by a non-creator", async () => {
    const input = {
      title: "Test Round Delete Error",
      location: "Test Location",
      eventType: "Test Event",
      date: "2024-12-01",
      time: "12:00:00",
      creatorID: "creator-id",
      discordID: "discord-id", // Ensure this is correctly provided
    };

    const round = await service.scheduleRound(input);

    await expect(
      service.deleteRound(round.roundID, "non-creator-id")
    ).rejects.toThrow("Only the creator can delete the round");
  });
});
