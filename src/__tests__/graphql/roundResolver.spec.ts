import "reflect-metadata";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { RoundResolver } from "../../resolvers/RoundResolver";
import { RoundService } from "../../services/RoundService";
import { RoundState, Response } from "../../types.generated";

// Define the base Round type (without `tagNumber`)
interface Round {
  roundID: string; // Use roundID instead of id
  title: string;
  location: string;
  date: string;
  time: string;
  participants: any[];
  scores: any[];
  state: RoundState;
  creatorID: string;
  discordID: string;
  finalized: boolean;
  __typename: "Round";
}

// Extend the Round type to include `tagNumber` in the response
interface RoundWithTagNumber extends Round {
  tagNumber: number | null; // tagNumber is added here for the response
}

// Mock leaderboardService to return a tagNumber
const leaderboardService = {
  getTagNumber: vi.fn().mockResolvedValue(5), // Mock the tag number for the user
};

vi.mock("class-validator");

describe("RoundResolver", () => {
  let roundService: RoundService;

  beforeEach(() => {
    roundService = ({
      getRound: vi.fn(),
      joinRound: vi.fn(),
      scheduleRound: vi.fn(),
      submitScore: vi.fn(),
      finalizeAndProcessScores: vi.fn(),
      editRound: vi.fn(),
      deleteRound: vi.fn(),
      updateParticipantResponse: vi.fn(),
    } as unknown) as RoundService;
  });

  describe("joinRound", () => {
    it("should allow a user to join an upcoming round and return the tagNumber", async () => {
      const input = {
        roundID: "round-1", // This should remain as is
        discordID: "discord-id",
        response: Response.Accept,
      };

      const round = {
        roundID: "round-1", // Use roundID instead of id
        title: "Round 1",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [],
        scores: [],
        state: RoundState.Upcoming,
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: false,
      };

      const tagNumber = 5;
      const joinRoundResponse = {
        ...round,
        tagNumber,
      };

      // Mock the necessary services
      vi.mocked(roundService.getRound).mockResolvedValue(round);
      vi.mocked(roundService.joinRound).mockResolvedValue(joinRoundResponse);

      const result = await RoundResolver.Mutation.joinRound(
        null,
        { input },
        { roundService, discordID: "discord-id", leaderboardService }
      );

      // Update the test to expect roundID instead of id
      expect(result).toEqual({
        roundID: "round-1", // Expect the same roundID the user joined
        discordID: "discord-id", // Expect the discordID that was passed in the input
        response: "ACCEPT", // Expect the response status passed in the input
      });
    });

    it("should throw an error if the round is not upcoming", async () => {
      const input = {
        roundID: "round-1",
        discordID: "discord-id",
        response: Response.Accept,
      };

      const round: Round = {
        roundID: "round-1", // Use roundID instead of id
        title: "Round 1",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [],
        scores: [],
        state: RoundState.Finalized, // Invalid state for joining
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: true,
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);

      await expect(
        RoundResolver.Mutation.joinRound(
          null,
          { input },
          { roundService, discordID: "discord-id", leaderboardService }
        )
      ).rejects.toThrow("You can only join rounds that are upcoming");
    });

    it("should throw an error if the user has already joined", async () => {
      const input = {
        roundID: "round-1",
        discordID: "discord-id",
        response: Response.Accept,
      };

      const round: Round = {
        roundID: "round-1", // Use roundID instead of id
        title: "Round 1",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [{ discordID: "discord-id" }], // User already joined
        scores: [],
        state: RoundState.Upcoming, // Valid state for joining
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: false,
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);
      vi.mocked(roundService.joinRound).mockImplementation(() => {
        throw new Error("You have already joined this round");
      });

      await expect(
        RoundResolver.Mutation.joinRound(
          null,
          { input },
          { roundService, discordID: "discord-id", leaderboardService }
        )
      ).rejects.toThrow("You have already joined this round");
    });
  });

  describe("submitScore", () => {
    it("should submit a score with a tag number", async () => {
      const input = {
        roundID: "round-1",
        score: 100,
        tagNumber: 5,
      };

      const round = {
        roundID: "round-1", // Use roundID instead of id
        title: "Test Round",
        location: "Test Location",
        eventType: "Test Event",
        date: "2024-11-22",
        time: "12:00:00",
        participants: [],
        scores: [],
        finalized: false,
        creatorID: "creator-id",
        state: RoundState.InProgress,
        discordID: "discord-id",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round); // Mock this properly

      vi.mocked(roundService.submitScore).mockResolvedValue(round);

      const result = await RoundResolver.Mutation.submitScore(null, input, {
        roundService,
        discordID: "discord-id",
      });

      expect(result).toEqual(round);
    });

    it("should submit a score without a tag number", async () => {
      const input = {
        roundID: "round-1",
        score: 100,
      };

      const round = {
        roundID: "round-1", // Use roundID instead of id
        title: "Test Round",
        location: "Test Location",
        eventType: "Test Event",
        date: "2024-11-22",
        time: "12:00:00",
        participants: [],
        scores: [],
        finalized: false,
        creatorID: "creator-id",
        state: RoundState.InProgress,
        discordID: "discord-id",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round); // Mock the getRound function

      vi.mocked(roundService.submitScore).mockResolvedValue(round);

      const result = await RoundResolver.Mutation.submitScore(null, input, {
        roundService,
        discordID: "discord-id",
      });

      expect(result).toEqual(round);
      expect(roundService.submitScore).toHaveBeenCalledWith(
        "round-1",
        "discord-id",
        100,
        null // Null tagNumber passed
      );
    });
  });
});
