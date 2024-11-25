import "reflect-metadata";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { RoundResolver } from "../../modules/round/round.resolver";
import { RoundService } from "../../modules/round/round.service";
import { RoundState, Response } from "../../types.generated"; // Ensure Response is imported correctly

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

// Mock leaderboardService to return a tagNumber
const leaderboardService = {
  getTagNumber: vi.fn().mockResolvedValue(5), // Mock the tag number for the user
};

vi.mock("class-validator");

describe("RoundResolver", () => {
  let roundService: RoundService;
  let roundResolver: RoundResolver;

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

    roundResolver = new RoundResolver(roundService, leaderboardService); // Instantiate the resolver
  });

  describe("joinRound", () => {
    it("should allow a user to join an upcoming round and return the tagNumber", async () => {
      const input: {
        roundID: string;
        discordID: string;
        response: Response;
      } = {
        roundID: "round-1",
        discordID: "discord-id",
        response: "ACCEPT", // Ensure this matches the Response type
      };

      const round: Round = {
        roundID: "round-1",
        title: "Round 1 ",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [],
        scores: [],
        state: "UPCOMING",
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: false,
        __typename: "Round",
      };

      const tagNumber = 5;
      const joinRoundResponse = {
        ...round,
        tagNumber,
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);
      vi.mocked(roundService.joinRound).mockResolvedValue(joinRoundResponse);

      const result = await roundResolver.joinRound(input, {
        roundService,
        discordID: "discord-id",
      });

      expect(result).toEqual({
        roundID: "round-1",
        discordID: "discord-id",
        response: "ACCEPT",
      });
    });

    it("should throw an error if the round is not upcoming", async () => {
      const input: {
        roundID: string;
        discordID: string;
        response: Response;
      } = {
        roundID: "round-1",
        discordID: "discord-id",
        response: "ACCEPT", // Ensure this matches the Response type
      };

      const round: Round = {
        roundID: "round-1",
        title: "Round 1",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [],
        scores: [],
        state: "FINALIZED", // Invalid state for joining
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: true,
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);

      await expect(
        roundResolver.joinRound(input, {
          roundService,
          discordID: "discord-id",
        })
      ).rejects.toThrow("You can only join rounds that are upcoming");
    });

    it("should throw an error if the user has already joined", async () => {
      const input: {
        roundID: string;
        discordID: string;
        response: Response;
      } = {
        roundID: "round-1",
        discordID: "discord-id",
        response: "ACCEPT", // Ensure this matches the Response type
      };

      const round: Round = {
        roundID: "round-1",
        title: "Round 1",
        location: "New York",
        date: "2024-12-01",
        time: "10:00",
        participants: [{ discordID: "discord-id" }], // User already joined
        scores: [],
        state: "UPCOMING",
        creatorID: "discord-id",
        discordID: "discord-id",
        finalized: false,
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);

      await expect(
        roundResolver.joinRound(input, {
          roundService,
          discordID: "discord-id",
        })
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

      const round: Round = {
        roundID: "round-1",
        title: "Test Round",
        location: "Test Location",
        date: "2024-11-22",
        time: "12:00:00",
        participants: [],
        scores: [],
        finalized: false,
        creatorID: "creator-id",
        state: "IN_PROGRESS",
        discordID: "discord-id",
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);
      vi.mocked(roundService.submitScore).mockResolvedValue(round);

      const result = await roundResolver.submitScore(
        {
          roundService,
          discordID: "discord-id",
        },
        input.roundID,
        input.score,
        input.tagNumber
      );

      expect(result).toEqual(round);
    });

    it("should submit a score without a tag number", async () => {
      const input = {
        roundID: "round-1",
        score: 100,
      };

      const round: Round = {
        roundID: "round-1",
        title: "Test Round",
        location: "Test Location",
        date: "2024-11-22",
        time: "12:00:00",
        participants: [],
        scores: [],
        finalized: false,
        creatorID: "creator-id",
        state: "IN_PROGRESS",
        discordID: "discord-id",
        __typename: "Round",
      };

      vi.mocked(roundService.getRound).mockResolvedValue(round);
      vi.mocked(roundService.submitScore).mockResolvedValue(round);

      const result = await roundResolver.submitScore(
        {
          roundService,
          discordID: "discord-id",
        }, // This is the context
        input.roundID, // This is the roundID
        input.score // This is the score
      );

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
