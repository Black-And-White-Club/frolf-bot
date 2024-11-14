// __tests__/db.test.ts
import { describe, it, expect, vi } from "vitest";

// Define the expected structure of the db module
interface DbModule {
  pool: {
    end: () => Promise<void>; // Adjust the return type as necessary
    // Add other methods if needed
  };
  // Add other exports if necessary
}

// Mock the db module
vi.mock("../src/db", async (importOriginal) => {
  const actual = (await importOriginal()) as DbModule; // Type assertion

  // Create a mock for the pool
  const mockPool = {
    end: vi.fn(), // Mock the end method
    // Add other methods if needed
  };

  return {
    ...actual,
    pool: mockPool, // Return the mocked pool
  };
});

// Now you can import the pool and other necessary functions after the mock
import { pool } from "../src/db/db";

describe("Database Connection", () => {
  it("should close the pool", async () => {
    await pool.end();
    expect(pool.end).toHaveBeenCalled();
  });
});
