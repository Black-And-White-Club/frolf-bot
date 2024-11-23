import "reflect-metadata"; // Import reflect-metadata at the top of your test file
import { describe, it, expect, beforeEach, vi } from "vitest";
import { ScoreResolver } from "../../resolvers/ScoreResolver"; // Adjust the path as necessary
import { ScoreService } from "../../services/ScoreService"; // Adjust the path as necessary
import { validate } from "class-validator"; // Assuming you're using class-validator for validation

// Define the Score type
interface Score {
  discordID: string;
  score: number;
  tagNumber: number;
  __typename: "Score"; // Assuming __typename is a literal type
}

vi.mock("class-validator");

describe("ScoreResolver", () => {
  let scoreService: ScoreService;

  beforeEach(() => {
    // Mock ScoreService methods
    scoreService = ({
      getUserScore: vi.fn(),
      getScoresForRound: vi.fn(),
      updateScore: vi.fn(),
      processScores: vi.fn(),
    } as unknown) as ScoreService;

    // Default mock behavior: no validation errors
    vi.mocked(validate).mockResolvedValue([]);
  });

  describe("getUser Score", () => {
    it("should return a score for the given discordID and roundID", async () => {
      const discordID = "user1";
      const roundID = "round-id";
      const expectedScore: Score = {
        discordID,
        score: -5,
        tagNumber: 123,
        __typename: "Score",
      };

      vi.mocked(scoreService.getUserScore).mockResolvedValue(expectedScore);

      const result = await ScoreResolver.Query.getUserScore(
        null,
        { discordID, roundID },
        { scoreService }
      );

      expect(result).toEqual(expectedScore);
      expect(scoreService.getUserScore).toHaveBeenCalledWith(
        discordID,
        roundID
      );
    });

    it("should throw an error when no score is found", async () => {
      const discordID = "user1";
      const roundID = "round-id";

      vi.mocked(scoreService.getUserScore).mockResolvedValue(null);

      await expect(
        ScoreResolver.Query.getUserScore(
          null,
          { discordID, roundID },
          { scoreService }
        )
      ).rejects.toThrow(
        "Score not found for the provided discordID and roundID"
      );
    });
  });

  describe("getScoresForRound", () => {
    it("should return scores for the given roundID", async () => {
      const roundID = "round-id";
      const expectedScores: Score[] = [
        { discordID: "user1", score: +5, tagNumber: 123, __typename: "Score" },
        { discordID: "user2", score: -3, tagNumber: 456, __typename: "Score" },
      ];

      vi.mocked(scoreService.getScoresForRound).mockResolvedValue(
        expectedScores
      );

      const result = await ScoreResolver.Query.getScoresForRound(
        null,
        { roundID },
        { scoreService }
      );

      expect(result).toEqual(expectedScores);
      expect(scoreService.getScoresForRound).toHaveBeenCalledWith(roundID);
    });
  });

  describe("updateScore", () => {
    it("should update an existing score", async () => {
      const input = {
        roundID: "round-id",
        discordID: "user1",
        score: +9,
        tagNumber: 123,
      };

      const updatedScore: Score = { ...input, __typename: "Score" };

      vi.mocked(scoreService.updateScore).mockResolvedValue(updatedScore);

      const result = await ScoreResolver.Mutation.updateScore(
        null,
        { input },
        { scoreService }
      );

      expect(result).toEqual(updatedScore);
      expect(scoreService.updateScore).toHaveBeenCalledWith(
        input.roundID,
        input.discordID,
        input.score,
        input.tagNumber
      );
    });

    it("should throw an error when updating a non-existing score", async () => {
      const input = {
        roundID: "round-id",
        discordID: "user1",
        score: +32,
        tagNumber: 123,
      };

      vi.mocked(scoreService.updateScore).mockRejectedValue(
        new Error("Score not found")
      );

      await expect(
        ScoreResolver.Mutation.updateScore(null, { input }, { scoreService })
      ).rejects.toThrow("Score not found");
    });
  });

  describe("processScores", () => {
    it("should process scores successfully", async () => {
      const input = {
        roundID: "round-id",
        scores: [
          { discordID: "user1", score: 100, tagNumber: 123 },
          { discordID: "user2", score: -20, tagNumber: 456 },
        ],
      };

      const expectedScores: Score[] = [
        {
          discordID: "user1",
          score: 100,
          tagNumber: 123,
          __typename: "Score",
        },
        { discordID: "user2", score: -20, tagNumber: 456, __typename: "Score" },
      ];

      vi.mocked(scoreService.processScores).mockResolvedValue(expectedScores);

      const result = await ScoreResolver.Mutation.processScores(
        null,
        { input },
        { scoreService }
      );

      expect(result).toEqual(expectedScores);
      expect(scoreService.processScores).toHaveBeenCalledWith(
        input.roundID,
        input.scores.map((score) => ({
          ...score,
          score: score.score,
        }))
      );
    });

    it("should throw an error on validation failure", async () => {
      const input = {
        roundID: "round-id",
        scores: [{ discordID: "user1", score: -10, tagNumber: null }],
      };

      vi.mocked(validate).mockResolvedValue([
        {
          property: "tagNumber",
          constraints: { isNotEmpty: "tagNumber should not be empty" },
        },
      ]);

      await expect(
        ScoreResolver.Mutation.processScores(null, { input }, { scoreService })
      ).rejects.toThrow("Validation failed!");
    });
  });
});
