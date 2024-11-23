import "reflect-metadata";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { LeaderboardResolver } from "../../resolvers/LeaderboardResolver";
import { LeaderboardService } from "../../services/LeaderboardService";
import { TagNumber } from "../../types.generated"; // Adjust the import based on your actual structure

describe("LeaderboardResolver", () => {
  let leaderboardService: LeaderboardService;

  beforeEach(() => {
    // Initialize the mock leaderboardService
    leaderboardService = ({
      getLeaderboard: vi.fn(),
      getUserTag: vi.fn(),
      updateTag: vi.fn(),
      processScores: vi.fn(),
    } as unknown) as LeaderboardService;
  });

  describe("getLeaderboard", () => {
    it("should return the leaderboard with the specified page and limit", async () => {
      const page = 1;
      const limit = 50;
      const mockUsers: TagNumber[] = [
        {
          __typename: "TagNumber",
          discordID: "user1",
          tagNumber: 1,
          lastPlayed: "",
          durationHeld: 0,
        },
        {
          __typename: "TagNumber",
          discordID: "user2",
          tagNumber: 2,
          lastPlayed: "",
          durationHeld: 0,
        },
      ];

      leaderboardService.getLeaderboard = vi
        .fn()
        .mockResolvedValue({ users: mockUsers });

      const result = await LeaderboardResolver.Query.getLeaderboard(
        null,
        { page, limit },
        { leaderboardService }
      );

      expect(result).toEqual({ users: mockUsers });
      expect(leaderboardService.getLeaderboard).toHaveBeenCalledWith(
        page,
        limit
      );
    });
  });

  describe("getUser Tag", () => {
    it("should return the tag for a valid discordID", async () => {
      const discordID = "user1";
      const mockTag: TagNumber = {
        __typename: "TagNumber",
        discordID,
        tagNumber: 1,
        lastPlayed: "",
        durationHeld: 0,
      };

      leaderboardService.getUserTag = vi.fn().mockResolvedValue(mockTag);

      const result = await LeaderboardResolver.Query.getUserTag(
        null,
        { discordID },
        { leaderboardService }
      );

      expect(result).toEqual(mockTag);
      expect(leaderboardService.getUserTag).toHaveBeenCalledWith(discordID);
    });

    it("should throw an error if the tag is not found", async () => {
      const discordID = "nonexistent-user";

      leaderboardService.getUserTag = vi.fn().mockResolvedValue(null);

      await expect(
        LeaderboardResolver.Query.getUserTag(
          null,
          { discordID },
          { leaderboardService }
        )
      ).rejects.toThrow("Tag not found for the provided discordID");
    });
  });

  describe("updateTag", () => {
    it("should update the tag for a valid discordID", async () => {
      const discordID = "user1";
      const tagNumber = 2;
      const mockTag: TagNumber = {
        __typename: "TagNumber",
        discordID,
        tagNumber,
        lastPlayed: "",
        durationHeld: 0,
      };

      leaderboardService.updateTag = vi.fn().mockResolvedValue(mockTag);

      const result = await LeaderboardResolver.Mutation.updateTag(
        null,
        { discordID, tagNumber },
        { leaderboardService }
      );

      expect(result).toEqual(mockTag);
      expect(leaderboardService.updateTag).toHaveBeenCalledWith(
        discordID,
        tagNumber
      );
    });

    it("should throw a validation error if the input is invalid", async () => {
      const discordID = "user1";
      const tagNumber = -1; // Assuming negative numbers are invalid

      await expect(
        LeaderboardResolver.Mutation.updateTag(
          null,
          { discordID, tagNumber },
          { leaderboardService }
        )
      ).rejects.toThrow("Validation failed!");
    });
  });

  describe("receiveScores", () => {
    it("should process scores and return the processed tags", async () => {
      const scores = [
        { discordID: "user1", score: -5, tagNumber: 1 },
        { discordID: "user2", score: -7, tagNumber: 2 },
      ];
      const processedTags = [
        {
          __typename: "TagNumber",
          discordID: "user1",
          tagNumber: 2,
          lastPlayed: "",
          durationHeld: 0,
        },
        {
          __typename: "TagNumber",
          discordID: "user2",
          tagNumber: 1,
          lastPlayed: "",
          durationHeld: 0,
        },
      ];

      leaderboardService.processScores = vi
        .fn()
        .mockResolvedValue(processedTags);

      const result = await LeaderboardResolver.Mutation.receiveScores(
        null,
        { scores },
        { leaderboardService }
      );

      expect(result).toEqual(processedTags);
      expect(leaderboardService.processScores).toHaveBeenCalledWith(scores);
    });

    it("should handle valid scores correctly", async () => {
      const scores = [
        { discordID: "user1", score: 10, tagNumber: 1 },
        { discordID: "user2", score: 20, tagNumber: 2 },
      ];
      const processedTags = [
        {
          __typename: "TagNumber",
          discordID: "user1",
          tagNumber: 1,
          lastPlayed: "",
          durationHeld: 0,
        },
        {
          __typename: "TagNumber",
          discordID: "user2",
          tagNumber: 2,
          lastPlayed: "",
          durationHeld: 0,
        },
      ];

      leaderboardService.processScores = vi
        .fn()
        .mockResolvedValue(processedTags);

      const result = await LeaderboardResolver.Mutation.receiveScores(
        null,
        { scores },
        { leaderboardService }
      );

      expect(result).toEqual(processedTags);
      expect(leaderboardService.processScores).toHaveBeenCalledWith(scores);
    });

    it("should not throw an error if a user does not have a score", async () => {
      const scores = [
        { discordID: "user1", score: -5, tagNumber: 1 },
        { discordID: "user2", score: 0 }, // No tagNumber provided
      ];
      const processedTags = [
        {
          __typename: "TagNumber",
          discordID: "user1",
          tagNumber: 2,
          lastPlayed: "",
          durationHeld: 0,
        },
      ];

      leaderboardService.processScores = vi
        .fn()
        .mockResolvedValue(processedTags);

      const result = await LeaderboardResolver.Mutation.receiveScores(
        null,
        { scores },
        { leaderboardService }
      );

      expect(result).toEqual(processedTags);
      expect(leaderboardService.processScores).toHaveBeenCalledWith(scores);
    });
  });

  describe("manualTagUpdate", () => {
    it("should manually update the tag for a valid discordID", async () => {
      const discordID = "user1";
      const newTagNumber = 3;
      const mockTag: TagNumber = {
        __typename: "TagNumber",
        discordID,
        tagNumber: newTagNumber,
        lastPlayed: "",
        durationHeld: 0,
      };

      leaderboardService.updateTag = vi.fn().mockResolvedValue(mockTag);

      const result = await LeaderboardResolver.Mutation.manualTagUpdate(
        null,
        { discordID, newTagNumber },
        { leaderboardService }
      );

      expect(result).toEqual(mockTag);
      expect(leaderboardService.updateTag).toHaveBeenCalledWith(
        discordID,
        newTagNumber
      );
    });

    it("should throw a validation error if the new tag number is invalid", async () => {
      const discordID = "user1";
      const newTagNumber = -1; // Assuming negative numbers are invalid

      await expect(
        LeaderboardResolver.Mutation.manualTagUpdate(
          null,
          { discordID, newTagNumber },
          { leaderboardService }
        )
      ).rejects.toThrow("Validation failed!");
    });
  });

  describe("Edge Cases", () => {
    it("should handle the case where all users have the same score", async () => {
      const scores = [
        { discordID: "user1", score: 10, tagNumber: 1 },
        { discordID: "user2", score: 10, tagNumber: 2 },
      ];
      const processedTags = [
        {
          __typename: "TagNumber",
          discordID: "user1",
          tagNumber: 1,
          lastPlayed: "",
          durationHeld: 0,
        },
        {
          __typename: "TagNumber",
          discordID: "user2",
          tagNumber: 2,
          lastPlayed: "",
          durationHeld: 0,
        },
      ];

      leaderboardService.processScores = vi
        .fn()
        .mockResolvedValue(processedTags);

      const result = await LeaderboardResolver.Mutation.receiveScores(
        null,
        { scores },
        { leaderboardService }
      );

      expect(result).toEqual(processedTags);
      expect(leaderboardService.processScores).toHaveBeenCalledWith(scores);
    });

    it("should handle an empty scores array without throwing an error", async () => {
      const scores: {
        discordID: string;
        score: number;
        tagNumber?: number;
      }[] = [];

      leaderboardService.processScores = vi.fn().mockResolvedValue([]);

      const result = await LeaderboardResolver.Mutation.receiveScores(
        null,
        { scores },
        { leaderboardService }
      );

      expect(result).toEqual([]);
      expect(leaderboardService.processScores).toHaveBeenCalledWith(scores);
    });
  });
});
