// src/__tests__/resolvers.test.ts
import { expect, test, vi, beforeEach } from "vitest";
import resolvers from "../src/resolvers/resolvers"; // Adjust the import based on your actual file structure
import { db } from "../src/db/db";

// Mock the database client
vi.mock("../src/db", () => ({
  db: {
    select: vi.fn().mockReturnThis(),
    from: vi.fn().mockReturnThis(),
    where: vi.fn().mockReturnThis(),
    execute: vi.fn(), // This should be the method you are calling in your resolvers
  },
}));

// Ensure that execute is a mock function
const mockExecute = vi.fn();
db.execute = mockExecute;

beforeEach(() => {
  vi.clearAllMocks(); // Clear any previous mock calls and results
});

test("should return user data", async () => {
  // Arrange
  const discordID = "1";
  const mockUser = { id: "1", discordID: "1", name: "John Doe", role: "user" };

  // Set up the mock to return the expected user data
  mockExecute.mockResolvedValueOnce([mockUser]); // Return an array with the mock user

  // Act
  const result = await resolvers.Query.get(null, { discordID });

  // Debugging output
  console.log("Mock Execute Calls:", mockExecute.mock.calls);
  console.info("Result from resolver:", result);

  // Assert
  expect(result).toEqual(mockUser);
});

test("should return null if user not found", async () => {
  // Arrange
  const discordID = "nonexistent";

  // Set up the mock to return no user data
  mockExecute.mockResolvedValueOnce([]); // Return an empty array

  // Act
  const result = await resolvers.Query.get(null, { discordID });

  // Assert
  expect(result).toBeNull();
});
