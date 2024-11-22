import { describe, it, expect, beforeEach, vi } from "vitest";
import { UserResolver } from "../../resolvers/UserResolver";
import { UserService } from "../../services/UserService";
import { UserRole } from "../../enums/user-role.enum";
import { User as UserType } from "../../types.generated";
import { validate } from "class-validator";

vi.mock("class-validator");

describe("UserResolver", () => {
  let userService: UserService;

  beforeEach(() => {
    // Mock UserService methods
    userService = ({
      createUser: vi.fn(),
      updateUser: vi.fn(),
      getUserByDiscordID: vi.fn(),
      getUserByTagNumber: vi.fn(),
    } as unknown) as UserService;

    // Default mock behavior: no validation errors
    vi.mocked(validate).mockResolvedValue([]);
  });

  describe("createUser", () => {
    it("should create a user successfully", async () => {
      const input = {
        name: "New User",
        discordID: "user-discord-id",
        tagNumber: 123,
        role: UserRole.Rattler,
      };
      const expectedUser: UserType = {
        __typename: "User",
        ...input,
      };

      // Mock createUser to resolve the expected user
      vi.mocked(userService.createUser).mockResolvedValue(expectedUser);

      const result = await UserResolver.Mutation.createUser(
        null,
        { input },
        { userService }
      );

      expect(result).toEqual(expectedUser);
      expect(userService.createUser).toHaveBeenCalledWith(input);
    });

    it("should throw an error if creating a user with an already taken tagNumber", async () => {
      const input = {
        name: "New User",
        discordID: "new-user-discord-id",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      // Mock userService.createUser to throw an error for duplicate tagNumber
      vi.mocked(userService.createUser).mockRejectedValue(
        new Error("Tag number already exists")
      );

      await expect(
        UserResolver.Mutation.createUser(null, { input }, { userService })
      ).rejects.toThrow("Tag number already exists");
    });

    it("should throw an error if user with the same discordID already exists", async () => {
      const input = {
        name: "New User",
        discordID: "existing-user-id",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      // Mock userService to reject with a specific error
      vi.mocked(userService.createUser).mockRejectedValue(
        new Error("User already exists")
      );

      await expect(
        UserResolver.Mutation.createUser(null, { input }, { userService })
      ).rejects.toThrow("User already exists");
    });

    it("should throw an error on validation failure (missing name)", async () => {
      const input = {
        name: "", // Invalid name
        discordID: "discord-id",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      vi.mocked(validate).mockResolvedValue([
        {
          property: "name",
          constraints: { isNotEmpty: "name should not be empty" },
        },
      ]);

      await expect(
        UserResolver.Mutation.createUser(null, { input }, { userService })
      ).rejects.toThrow("Validation failed!");
    });
  });

  describe("updateUser", () => {
    it("should update a user successfully by Admin", async () => {
      const input = {
        discordID: "user-discord-id",
        name: "Updated User",
        tagNumber: 456,
        role: UserRole.Editor, // Explicitly provide role
      };

      const existingUser = {
        discordID: "user-discord-id",
        name: "Existing User",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      const expectedUser: UserType = {
        __typename: "User",
        ...input,
      };

      // Mock getUserByDiscordID to return the existing user
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(existingUser);

      // Mock updateUser to resolve the updated user
      vi.mocked(userService.updateUser).mockResolvedValue(expectedUser);

      const result = await UserResolver.Mutation.updateUser(
        null,
        { input, requesterRole: UserRole.Admin },
        { userService }
      );

      expect(result).toEqual(expectedUser);
      expect(userService.updateUser).toHaveBeenCalledWith(
        input,
        UserRole.Admin
      );
    });

    it("should throw an error if updating with a duplicate tagNumber", async () => {
      const existingUser = {
        discordID: "user-discord-id",
        name: "Existing User",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      const input = {
        discordID: existingUser.discordID,
        name: "Updated User",
        tagNumber: existingUser.tagNumber, // Duplicate tagNumber
        role: UserRole.Editor,
      };

      // Mock getUserByDiscordID to return the existing user
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(existingUser);

      // Mock userService to reject with a specific error
      vi.mocked(userService.updateUser).mockRejectedValue(
        new Error("Duplicate tagNumber")
      );

      await expect(
        UserResolver.Mutation.updateUser(
          null,
          { input, requesterRole: UserRole.Admin },
          { userService }
        )
      ).rejects.toThrow("Duplicate tagNumber");
    });
    it("should not update role if not provided", async () => {
      const existingUser = {
        discordID: "user-discord-id",
        name: "Existing User",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      const input = {
        discordID: existingUser.discordID,
        name: "Updated User Without Role Change",
        tagNumber: 456,
        role: existingUser.role, // Include role explicitly
      };

      const expectedUser: UserType = {
        __typename: "User",
        discordID: existingUser.discordID,
        name: input.name,
        tagNumber: input.tagNumber,
        role: existingUser.role,
      };

      // Mock getUserByDiscordID to return the existing user
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(existingUser);

      // Mock updateUser to resolve the updated user
      vi.mocked(userService.updateUser).mockResolvedValue(expectedUser);

      const result = await UserResolver.Mutation.updateUser(
        null,
        { input, requesterRole: UserRole.Admin },
        { userService }
      );

      expect(result).toEqual(expectedUser);

      expect(userService.updateUser).toHaveBeenCalledWith(
        {
          ...input,
          role: existingUser.role, // Add role from existing user
        },
        UserRole.Admin
      );
    });
  });

  describe("getUser", () => {
    it("should throw an error if user not found by discordID", async () => {
      const discordID = "non-existent-user";

      // Simulate a userService where no user is found
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(null);

      // Expect the promise to reject with the error "User not found!"
      await expect(
        UserResolver.Query.getUser(null, { discordID }, { userService })
      ).rejects.toThrow("User not found by discordID");
    });

    it("should return a user by discordID", async () => {
      const discordID = "existing-user";
      const user = {
        discordID,
        name: "User",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      // Simulate a userService where a user is found
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(user);

      // Expect the result to be the user object
      const result = await UserResolver.Query.getUser(
        null,
        { discordID },
        { userService }
      );

      expect(result).toEqual(user);
    });

    it("should throw an error if user not found by Tag Number", async () => {
      const tagNumber = 1234;

      // Simulate a userService where no user is found
      vi.mocked(userService.getUserByTagNumber).mockResolvedValue(null);

      // Expect the promise to reject with the error "User not found by tagNumber"
      await expect(
        UserResolver.Query.getUser(null, { tagNumber }, { userService })
      ).rejects.toThrow("User not found by Tag");
    });

    it("should return a user by tagNumber", async () => {
      const tagNumber = 123;
      const user = {
        discordID: "existing-user",
        name: "User",
        tagNumber,
        role: UserRole.Rattler,
      };

      // Simulate a userService where a user is found
      vi.mocked(userService.getUserByTagNumber).mockResolvedValue(user);

      // Expect the result to be the user object
      const result = await UserResolver.Query.getUser(
        null,
        { tagNumber },
        { userService }
      );

      expect(result).toEqual(user);
    });
  });
});
