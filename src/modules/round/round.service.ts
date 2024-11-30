// src/modules/round/round.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { Cron, CronExpression } from "@nestjs/schedule";
import { eq, sql } from "drizzle-orm";
import { RoundModel } from "./round.model";
import { Round, Participant } from "./round.entity";
import { ScheduleRoundInput } from "src/dto/round/round-input.dto";
import { JoinRoundInput } from "src/dto/round/join-round-input.dto";
import { EditRoundInput } from "src/dto/round/edit-round-input.dto";
import { validate } from "class-validator";
import { NodePgDatabase } from "drizzle-orm/node-postgres";
import { RoundState, Response } from "src/enums";
import { Publisher } from "src/rabbitmq";

@Injectable()
export class RoundService {
  constructor(
    @Inject("ROUND_DATABASE_CONNECTION") private readonly db: NodePgDatabase,
    private readonly publisher: Publisher
  ) {}

  async getRounds(limit: number = 10, offset: number = 0): Promise<Round[]> {
    try {
      const rounds = await this.db
        .select()
        .from(RoundModel)
        .limit(limit)
        .offset(offset);
      return rounds.map((round) => this.mapRound(round));
    } catch (error) {
      console.error("Error fetching rounds:", error);
      throw new Error("Failed to fetch rounds");
    }
  }

  async getRound(roundID: string): Promise<Round | null> {
    try {
      const rounds = await this.db
        .select()
        .from(RoundModel)
        .where(eq(RoundModel.roundID, Number(roundID)));
      if (rounds.length > 0) {
        return this.mapRound(rounds[0]);
      }
      return null;
    } catch (error) {
      console.error("Error fetching round:", error);
      throw new Error("Failed to fetch round");
    }
  }

  async scheduleRound(input: ScheduleRoundInput): Promise<Round> {
    try {
      console.log("Input before validation:", input);

      if (
        !input.title ||
        !input.location ||
        !input.date ||
        !input.time ||
        !input.creatorID
      ) {
        throw new Error("Required fields are missing or undefined.");
      }

      const plainInput = JSON.parse(JSON.stringify(input));
      console.log("Plain Input after JSON parse:", plainInput);

      const errors = await validate(plainInput);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      const roundData = {
        title: plainInput.title,
        location: plainInput.location,
        eventType: plainInput.eventType || null,
        date: plainInput.date,
        time: plainInput.time,
        participants: JSON.stringify([]),
        scores: JSON.stringify([]),
        finalized: false,
        creatorID: plainInput.creatorID,
        state: RoundState.Upcoming, // Use the RoundState enum
      };

      const [round] = await this.db
        .insert(RoundModel)
        .values(roundData)
        .returning();

      return this.mapRound(round);
    } catch (error) {
      console.error("Error scheduling round:", error);
      throw new Error("Failed to schedule round");
    }
  }

  async joinRound(
    input: JoinRoundInput & { tagNumber: number | null }
  ): Promise<Round> {
    try {
      const { roundID, discordID, response, tagNumber } = input;

      // Access Response enum values through the Participant entity
      const validResponses = [
        Response.Accept,
        Response.Tentative,
        Response.Decline,
      ];
      if (!validResponses.includes(response)) {
        throw new Error(`Invalid response value: ${response}`);
      }

      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      if (round.state !== RoundState.Upcoming) {
        throw new Error("You can only join rounds that are upcoming");
      }

      const participants = [...round.participants];
      if (participants.find((p) => p.discordID === discordID)) {
        throw new Error("Participant already joined the round");
      }

      const participant: Participant = {
        discordID,
        response,
        tagNumber: tagNumber === null ? undefined : tagNumber,
      };
      participants.push(participant);

      await this.db
        .update(RoundModel)
        .set({ participants: JSON.stringify(participants) })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return { ...round, participants };
    } catch (error) {
      console.error("Error joining round:", error);
      throw new Error("Failed to join round");
    }
  }

  async submitScore(
    roundID: string,
    discordID: string,
    score: number,
    tagNumber: number | null
  ): Promise<Round> {
    try {
      const round = await this.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.state === RoundState.Finalized) {
        throw new Error("Cannot submit scores for a finalized round");
      }

      // Find the index of the existing score for the participant
      const scoreIndex = round.scores.findIndex(
        (s) => s.discordID === discordID
      );

      if (scoreIndex !== -1) {
        // Update the existing score
        round.scores[scoreIndex] = { discordID, score, tagNumber };
      } else {
        // Add a new score
        round.scores.push({ discordID, score, tagNumber });
      }

      // Update the round in the database
      await this.db
        .update(RoundModel)
        .set({ scores: JSON.stringify(round.scores) })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return { ...round };
    } catch (error) {
      console.error("Error submitting score:", error);
      throw new Error("Failed to submit score");
    }
  }

  async finalizeAndProcessScores(roundID: string): Promise<Round> {
    try {
      const round = await this.getRound(roundID);
      if (!round) {
        throw new Error("Round not found");
      }

      if (round.finalized) {
        throw new Error("Round has already been finalized");
      }

      // Publish the scores to RabbitMQ for processing
      await this.publisher.publishMessage("process_scores", {
        roundID,
        scores: round.scores,
      });

      // Update the round state to finalized
      round.state = RoundState.Finalized;
      round.finalized = true;

      await this.db
        .update(RoundModel)
        .set({ state: RoundState.Finalized, finalized: true })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return round;
    } catch (error) {
      console.error("Error finalizing and processing scores:", error);
      throw new Error("Failed to finalize and process scores");
    }
  }

  async editRound(roundID: string, input: EditRoundInput): Promise<Round> {
    try {
      // Validate the input using class-validator
      const plainInput = JSON.parse(JSON.stringify(input));
      const errors = await validate(plainInput);
      if (errors.length > 0) {
        console.error("Validation errors:", errors); // Log validation errors for debugging
        throw new Error("Validation failed!");
      }

      // Update the round directly using Drizzle ORM
      const [updatedRound] = await this.db
        .update(RoundModel)
        .set(input)
        .where(eq(RoundModel.roundID, Number(roundID)))
        .returning();

      return this.mapRound(updatedRound);
    } catch (error) {
      console.error("Error editing round:", error);
      throw new Error("Failed to edit round");
    }
  }

  async deleteRound(roundID: string, userID: string): Promise<boolean> {
    try {
      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");
      if (round.creatorID !== userID) {
        throw new Error("Only the creator can delete the round");
      }

      await this.db
        .delete(RoundModel)
        .where(eq(RoundModel.roundID, Number(roundID)));
      return true;
    } catch (error) {
      console.error("Error deleting round:", error);
      throw new Error("Failed to delete round");
    }
  }

  async updateParticipantResponse(
    roundID: string,
    discordID: string,
    response: Response
  ): Promise<Round> {
    // Use ResponseEnum here
    try {
      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      const participants = [...round.participants];
      const participant = participants.find((p) => p.discordID === discordID);
      if (!participant) throw new Error("Participant not found");

      participant.response = response;

      await this.db
        .update(RoundModel)
        .set({ participants: JSON.stringify(participants) })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return { ...round, participants };
    } catch (error) {
      console.error("Error updating participant response:", error);
      throw new Error("Failed to update participant response");
    }
  }

  private mapRound(round: any): Round {
    const safeParse = (jsonString: any, fallback: any) => {
      try {
        return typeof jsonString === "string"
          ? JSON.parse(jsonString)
          : jsonString;
      } catch (error) {
        console.error("Failed to parse JSON:", error, jsonString);
        return fallback;
      }
    };

    return {
      roundID: round.roundID.toString(),
      title: round.title,
      location: round.location,
      eventType: round.eventType,
      date: round.date,
      time: round.time,
      participants: safeParse(round.participants, []),
      scores: safeParse(round.scores, []),
      finalized: round.finalized,
      creatorID: round.creatorID,
      state: round.state as RoundState,
      createdAt: round.createdAt,
      updatedAt: round.updatedAt,
    };
  }

  @Cron(CronExpression.EVERY_MINUTE)
  async checkForUpcomingRounds() {
    const now = new Date();
    const oneHourFromNow = new Date(now.getTime() + 60 * 60 * 1000);

    const upcomingRounds = await this.db
      .select()
      .from(RoundModel)
      .where(
        sql`
          state = ${RoundState.Upcoming} AND 
          date = ${now.toISOString().split("T")[0]} AND 
          time BETWEEN ${now.toLocaleTimeString()} AND ${oneHourFromNow.toLocaleTimeString()}
        `
      );

    for (const round of upcomingRounds) {
      const roundID = round.roundID.toString();
      const startTime = new Date(`${round.date}T${round.time}`);
      const oneHourBefore = new Date(startTime.getTime() - 60 * 60 * 1000);

      if (now >= oneHourBefore && now < startTime) {
        console.log(`Sending 1-hour notification for round ${roundID}`);
        // ... (Call DiscordBotService to send the notification) ...
      }

      if (now >= startTime) {
        console.log(`Sending round start notification for round ${roundID}`);
        // ... (Call DiscordBotService to send the notification) ...

        round.state = RoundState.InProgress;
        await this.db
          .update(RoundModel)
          .set({ state: RoundState.InProgress })
          .where(eq(RoundModel.roundID, Number(roundID)));
      }
    }
  }
}
