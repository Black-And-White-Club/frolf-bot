import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { UserResolver } from "../../modules/user/user.resolver";
import { UserService } from "../../modules/user/user.service";
import { UserRole } from "../../enums/user-role.enum";
import { validate } from "class-validator";
import { UpdateUserDto } from "../../dto/user/update-user.dto";
import { CreateUserDto } from "../../dto/user/create-user.dto";

vi.mock("class-validator");

describe("UserResolver", () => {
  let userService;
  let userResolver;
  beforeEach(() => {
    userService = ({
      createUser: vi.fn(),
      updateUser: vi.fn(),
      getUserByDiscordID: vi.fn(),
      getUserByTagNumber: vi.fn(),
    } as unknown) as UserService;

    // Default mock behavior: no validation errors
    vi.mocked(validate).mockResolvedValue([]);
    userResolver = new UserResolver(userService);
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2024, 10, 24));
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  describe("createUser", () => {
    it("should create a user successfully", async () => {
      const input: CreateUserDto = {
        name: "New User",
        discordID: "user-discord-id",
        tagNumber: 123,
        role: UserRole.RATTLER,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      const expectedResult = {
        deletedAt: null,
        discordID: input.discordID,
        name: input.name,
        role: input.role,
        tagNumber: input.tagNumber,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      vi.mocked(userService.createUser).mockResolvedValue(expectedResult);

      const result = await userResolver.createUser({ input });

      expect(result).toEqual(expectedResult);
    });

    it("should throw an error if creating a user with an already taken tagNumber", async () => {
      const input = {
        name: "New User",
        discordID: "new-user-discord-id",
        tagNumber: 123,
        role: UserRole.RATTLER,
      };

      vi.mocked(userService.createUser).mockRejectedValue(
        new Error("Tag number already exists")
      );

      await expect(userResolver.createUser({ input })).rejects.toThrow(
        "Tag number already exists"
      );
    });

    it("should throw an error if user with the same discordID already exists", async () => {
      const input = {
        name: "New User",
        discordID: "existing-user-id",
        tagNumber: 123,
        role: UserRole.RATTLER,
      };

      vi.mocked(userService.createUser).mockRejectedValue(
        new Error("User  already exists")
      );

      await expect(userResolver.createUser({ input })).rejects.toThrow(
        "User  already exists"
      );
    });

    it("should throw an error on validation failure (missing name)", async () => {
      const input = {
        name: "",
        discordID: "discord-id",
        tagNumber: 123,
        role: UserRole.RATTLER,
      };

      vi.mocked(validate).mockResolvedValue([
        {
          property: "name",
          constraints: { isNotEmpty: "name should not be empty" },
        },
      ]);

      await expect(userResolver.createUser({ input })).rejects.toThrow(
        /Validation failed/
      );
    });
  });

  describe("updateUser", () => {
    it("should update a user successfully by Admin", async () => {
      const input: UpdateUserDto = {
        discordID: "user-discord-id",
        name: "Updated User",
        tagNumber: 456,
        role: UserRole.EDITOR,
      };

      const existingUser = {
        discordID: "user-discord-id",
        name: "Existing User",
        tagNumber: 123,
        role: UserRole.RATTLER,
      };

      const updatedUser = {
        ...existingUser,
        ...input,
        updatedAt: new Date().toISOString(),
      };

      userService.getUserByDiscordID.mockResolvedValue(existingUser);
      userService.updateUser.mockResolvedValue(updatedUser);

      const result = await userResolver.updateUser(input, UserRole.ADMIN);

      expect(result).toEqual(updatedUser);
    });

    it("should throw an error if updating with a duplicate tagNumber", async () => {
      const existingUser = {
        discordID: "user-discord-id",
        name: "Existing User",
        tagNumber: 123,
        role: UserRole.RATTLER,
      };

      const input = {
        discordID: existingUser.discordID,
        name: "Updated User",
        tagNumber: existingUser.tagNumber,
        role: UserRole.EDITOR,
      };

      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(existingUser);
      vi.mocked(userService.updateUser).mockRejectedValue(
        new Error("User with this tag number already exists")
      );

      await expect(
        userResolver.updateUser(input, UserRole.ADMIN)
      ).rejects.toThrow("User with this tag number already exists");
    });

    it("should throw an error if the user does not exist", async () => {
      const input = {
        discordID: "non-existing-user-id",
        name: "Updated User",
      };

      vi.mocked(userService.getUserByDiscordID).mockResolvedValue(null);

      await expect(
        userResolver.updateUser(input, UserRole.ADMIN)
      ).rejects.toThrow("User not found");
    });
  });
});
