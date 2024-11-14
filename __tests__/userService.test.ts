// src/__tests__/userService.test.ts
import { expect, test, vi } from "vitest";
import { UserServiceImpl } from "../src/service/userService"; // Ensure this is the correct import
import { Pool } from "pg"; // PostgreSQL client

// Mock the Pool class from pg
const mockQuery = vi.fn();
const mockClient = {
  query: mockQuery,
};

// Create an instance of UserServiceImpl with the mocked client
const userService = new UserServiceImpl(mockClient as unknown as Pool);

test("getUser ByDiscordID should return user data", async () => {
  // Arrange
  const discordID = "1";
  const mockUser = { id: "1", discordID: "1", name: "John Doe", role: "" };

  // Mock the database query response
  mockQuery.mockResolvedValueOnce({ rows: [mockUser] });

  // Act
  const user = await userService.getUserByDiscordID(discordID);

  // Assert
  expect(user).toEqual(mockUser);
});

test("getUser ByDiscordID should return null if user not found", async () => {
  // Arrange
  const discordID = "nonexistent";

  // Mock the database query response
  mockQuery.mockResolvedValueOnce({ rows: [] });

  // Act
  const user = await userService.getUserByDiscordID(discordID);

  // Assert
  expect(user).toBeNull();
});

test("createUser should create a new user", async () => {
  // Arrange
  const input = { discordID: "1", name: "John Doe", role: "user" }; // Include role
  const mockCreatedUser = {
    id: "1",
    discordID: "1",
    name: "John Doe",
    role: "user",
  };

  // Mock the database query response for user existence check
  mockQuery.mockResolvedValueOnce({ rows: [] }); // No existing user

  // Mock the database query response for user creation
  mockQuery.mockResolvedValueOnce({ rows: [mockCreatedUser] });

  // Act
  const createdUser = await userService.createUser(input);

  // Assert
  expect(createdUser).toEqual(mockCreatedUser);
});

test("createUser should throw an error if user already exists", async () => {
  // Arrange
  const input = { discordID: "1", name: "John Doe", role: "user" }; // Include role

  // Mock the database query response to simulate existing user
  mockQuery.mockResolvedValueOnce({
    rows: [{ id: "1", discordID: "1", name: "John Doe", role: "user" }],
  });

  // Act & Assert
  await expect(userService.createUser(input)).rejects.toThrow(
    `User with Discord ID ${input.discordID} already exists`
  );
});

test("createUser should throw an error if input is invalid", async () => {
  // Arrange
  const input = { discordID: "", name: "", role: "" }; // Invalid input

  // Act & Assert
  await expect(userService.createUser(input)).rejects.toThrow(
    "DiscordID and Name are required"
  );
});
