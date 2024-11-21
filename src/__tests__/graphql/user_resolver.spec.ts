// src/__tests__/graphql/user_resolver.spec.ts
import { describe, it, expect, beforeEach, vi } from "vitest";
import { UserResolver } from "../../resolvers/UserResolver";
import { UserService } from "../../services/UserService";
import { UserRole } from "../../enums/user-role.enum";
import { User as UserType } from "../../types.generated";
import { validate } from "class-validator";

vi.mock("class-validator");

describe("UserResolver", () => {
  let userService: UserService;
  let userResolver: UserResolver;

  beforeEach(() => {
    // Mock UserService methods
    userService = ({
      createUser: vi.fn(),
      updateUser: vi.fn(),
      getUserByDiscordID: vi.fn(),
    } as unknown) as UserService;

    userResolver = new UserResolver(userService);

    // Default: no validation errors
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

      const result = await userResolver.createUser(input);

      expect(result).toEqual(expectedUser);
      expect(userService.createUser).toHaveBeenCalledWith(input);
    });

    it("should throw an error on validation failure", async () => {
      const input = {
        name: "", // Invalid name
        discordID: "",
        tagNumber: 123,
        role: UserRole.Rattler,
      };

      // Mock validation to return errors
      vi.mocked(validate).mockResolvedValue([
        {
          property: "name",
          constraints: { isNotEmpty: "name should not be empty" },
        },
      ]);

      await expect(userResolver.createUser(input)).rejects.toThrow(
        "Validation failed!"
      );
    });
  });

  describe("updateUser", () => {
    it("should update a user successfully by Admin", async () => {
      const input = {
        discordID: "user-discord-id",
        name: "Updated User",
        tagNumber: 456,
        role: UserRole.Editor,
      };
      const expectedUser: UserType = {
        __typename: "User",
        ...input,
      };

      // Mock updateUser to resolve the expected user
      vi.mocked(userService.updateUser).mockResolvedValue(expectedUser);

      const result = await userResolver.updateUser(input, UserRole.Admin);

      expect(result).toEqual(expectedUser);
      expect(userService.updateUser).toHaveBeenCalledWith(
        input,
        UserRole.Admin
      );
    });

    it("should throw an error if non-Admin tries to update user role", async () => {
      const input = {
        discordID: "user-discord-id",
        name: "Updated User",
        tagNumber: 456,
        role: UserRole.Editor,
      };

      await expect(
        userResolver.updateUser(input, UserRole.Rattler)
      ).rejects.toThrow("Only Admin can update user roles to Editor or Admin.");
    });

    it("should throw an error on validation failure", async () => {
      const input = {
        discordID: "user-discord-id",
        name: "", // Invalid name
        tagNumber: 456,
        role: UserRole.Rattler,
      };

      // Mock validation to return errors
      vi.mocked(validate).mockResolvedValue([
        {
          property: "name",
          constraints: { isNotEmpty: "name should not be empty" },
        },
      ]);

      await expect(
        userResolver.updateUser(input, UserRole.Admin)
      ).rejects.toThrow("Validation failed!");
    });
  });

  describe("getUser", () => {
    it("should return a user by discordID", async () => {
      const discordID = "some-discord-id";
      const expectedUser: UserType = {
        __typename: "User",
        discordID,
        name: "Existing User",
        tagNumber: 789,
        role: UserRole.Rattler,
      };

      // Mock getUserByDiscordID to resolve the expected user
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(expectedUser);

      const result = await userResolver.getUser(discordID);

      expect(result).toEqual(expectedUser);
      expect(userService.getUserByDiscordID).toHaveBeenCalledWith(discordID);
    });

    it("should return null if user not found", async () => {
      const discordID = "non-existent-discord-id";

      // Mock getUserByDiscordID to return null
      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(null);

      const result = await userResolver.getUser(discordID);

      expect(result).toBeNull();
      expect(userService.getUserByDiscordID).toHaveBeenCalledWith(discordID);
    });
  });
});
