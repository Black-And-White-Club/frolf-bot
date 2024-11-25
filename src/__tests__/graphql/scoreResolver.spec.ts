import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { ScoreResolver } from "../../modules/score/score.resolver";
import { ScoreService } from "../../modules/score/score.service";
import { validate } from "class-validator";
import { UpdateScoreDto } from "../../dto/score/update-score.dto"; // Adjust the path as necessary
import { ProcessScoresDto } from "../../dto/score/process-scores.dto"; // Adjust the path as necessary

vi.mock("class-validator");

describe("ScoreResolver", () => {
  let scoreService: ScoreService;
  let scoreResolver: ScoreResolver;

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
    scoreResolver = new ScoreResolver(scoreService); // Create an instance of ScoreResolver
  });

  describe("getUserScore", () => {
    it("should return a score for the given discordID and roundID", async () => {
      const discordID = "user1";
      const roundID = "round-id";
      const expectedScore = {
        discordID,
        score: -5,
        tagNumber: 123,
        __typename: "Score" as const, // Ensure __typename is a literal type
      };

      vi.mocked(scoreService.getUserScore).mockResolvedValue(expectedScore);

      const result = await scoreResolver.getUserScore(discordID, roundID, {
        scoreService,
      });

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
        scoreResolver.getUserScore(discordID, roundID, { scoreService })
      ).rejects.toThrow(
        "Score not found for the provided discordID and roundID"
      );
    });
  });

  describe("getScoresForRound", () => {
    it("should return scores for the given roundID", async () => {
      const roundID = "round-id";
      const expectedScores = [
        {
          discordID: "user1",
          score: +5,
          tagNumber: 123,
          __typename: "Score" as const,
        },
        {
          discordID: "user2",
          score: -3,
          tagNumber: 456,
          __typename: "Score" as const,
        },
      ];

      vi.mocked(scoreService.getScoresForRound).mockResolvedValue(
        expectedScores
      );

      const result = await scoreResolver.getScoresForRound(roundID, {
        scoreService,
      });

      expect(result).toEqual(expectedScores);
      expect(scoreService.getScoresForRound).toHaveBeenCalledWith(roundID);
    });
  });

  describe("updateScore", () => {
    it("should update an existing score", async () => {
      const input: UpdateScoreDto = {
        roundID: "round-id",
        discordID: "user1",
        score: +9,
        tagNumber: 123,
      };

      const updatedScore = { ...input, __typename: "Score" as const };

      vi.mocked(scoreService.updateScore).mockResolvedValue(updatedScore);

      const result = await scoreResolver.updateScore(input); // Pass input directly

      expect(result).toEqual(updatedScore);
      expect(scoreService.updateScore).toHaveBeenCalledWith(
        input.roundID,
        input.discordID,
        input.score,
        input.tagNumber
      );
    });

    it("should throw an error when updating a non-existing score", async () => {
      const input: UpdateScoreDto = {
        roundID: "round-id",
        discordID: "user1",
        score: +32,
        tagNumber: 123,
      };

      vi.mocked(scoreService.updateScore).mockRejectedValue(
        new Error("Score not found")
      );

      await expect(
        scoreResolver.updateScore(input) // Pass input directly
      ).rejects.toThrow("Score not found");
    });
  });

  describe("processScores", () => {
    it("should process scores successfully", async () => {
      const input: ProcessScoresDto = {
        roundID: "round-id",
        scores: [
          { discordID: "user1", score: 100, tagNumber: 123 },
          { discordID: "user2", score: -20, tagNumber: 456 },
        ],
      };

      const expectedScores = [
        {
          discordID: "user1",
          score: 100,
          tagNumber: 123,
          __typename: "Score" as const,
        },
        {
          discordID: "user2",
          score: -20,
          tagNumber: 456,
          __typename: "Score" as const,
        },
      ];

      vi.mocked(scoreService.processScores).mockResolvedValue(expectedScores);

      const result = await scoreResolver.processScores(input); // Pass input directly

      expect(result).toEqual(expectedScores);
      expect(scoreService.processScores).toHaveBeenCalledWith(
        input.roundID,
        input.scores.map((score) => ({
          ...score,
          score: parseInt(score.score.toString(), 10), // Ensure score is an integer
        }))
      );
    });

    it("should throw an error on validation failure", async () => {
      const input: ProcessScoresDto = {
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
        scoreResolver.processScores(input) // Pass input directly
      ).rejects.toThrow("Validation failed:");
    });
  });
});
