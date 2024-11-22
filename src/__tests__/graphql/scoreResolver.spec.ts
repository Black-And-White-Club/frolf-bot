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

  describe("processScores", () => {
    it("should process scores successfully", async () => {
      const input = {
        roundID: "round-id",
        scores: [
          { discordID: "user1", score: 100, tagNumber: 123 },
          { discordID: "user2", score: 200, tagNumber: 456 },
        ],
      };

      // Mock the expected result as an array of Score objects
      const expectedScores: Score[] = [
        { discordID: "user1", score: 100, tagNumber: 123, __typename: "Score" },
        { discordID: "user2", score: 200, tagNumber: 456, __typename: "Score" },
      ];

      // Mock the processScores method to resolve the expected scores
      vi.mocked(scoreService.processScores).mockResolvedValue(expectedScores);

      const result = await ScoreResolver.Mutation.processScores(
        null,
        { input },
        { scoreService }
      );

      expect(result).toEqual(expectedScores);
      expect(scoreService.processScores).toHaveBeenCalledWith(
        input.roundID,
        input.scores
      );
    });

    it("should throw an error on validation failure", async () => {
      const input = {
        roundID: "round-id",
        scores: [
          { discordID: "user1", score: 100, tagNumber: null }, // Assuming tagNumber is required
        ],
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
