// src/modules/round/round.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { eq } from "drizzle-orm";
import { RoundModel } from "./round.model";
import {
  Round as GraphQLRound,
  Participant as GraphQLParticipant,
  RoundState,
  Response,
} from "../../types.generated";
import { ScheduleRoundInput } from "../../dto/round/round-input.dto";
import { JoinRoundInput } from "../../dto/round/join-round-input.dto";
import { ScoreService } from "../score/score.service";
import { EditRoundInput } from "../../dto/round/edit-round-input.dto";
import { validate } from "class-validator";
import { NodePgDatabase } from "drizzle-orm/node-postgres";

@Injectable()
export class RoundService {
  constructor(
    @Inject("DATABASE_CONNECTION") private readonly db: NodePgDatabase
  ) {}

  async getRounds(
    limit: number = 10,
    offset: number = 0
  ): Promise<GraphQLRound[]> {
    try {
      const rounds = await this.db
        .select()
        .from(RoundModel)
        .limit(limit)
        .offset(offset);

      return rounds.map((round) => this.mapRoundToGraphQL(round));
    } catch (error) {
      console.error("Error fetching rounds:", error);
      throw new Error("Failed to fetch rounds");
    }
  }

  async getRound(roundID: string): Promise<GraphQLRound | null> {
    try {
      const rounds = await this.db
        .select()
        .from(RoundModel)
        .where(eq(RoundModel.roundID, Number(roundID)));

      if (rounds.length > 0) {
        return this.mapRoundToGraphQL(rounds[0]);
      }
      return null;
    } catch (error) {
      console.error("Error fetching round:", error);
      throw new Error("Failed to fetch round");
    }
  }

  async scheduleRound(input: ScheduleRoundInput): Promise<GraphQLRound> {
    try {
      // Log input to check for issues
      console.log("Input before validation:", input);

      // Manually check if any required fields are undefined
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

      // Perform the validation
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
        participants: JSON.stringify([]), // Initialize as valid JSON array
        scores: JSON.stringify([]), // Initialize as valid JSON array
        finalized: false,
        creatorID: plainInput.creatorID,
        state: "UPCOMING",
      };

      const [round] = await this.db
        .insert(RoundModel)
        .values(roundData)
        .returning();

      return this.mapRoundToGraphQL(round);
    } catch (error) {
      console.error("Error scheduling round:", error);
      throw new Error("Failed to schedule round");
    }
  }

  async joinRound(
    input: JoinRoundInput & { tagNumber: number | null }
  ): Promise<GraphQLRound> {
    try {
      const { roundID, discordID, response, tagNumber } = input;

      const validResponses = ["ACCEPT", "TENTATIVE", "DECLINE"];
      if (!validResponses.includes(response)) {
        throw new Error(`Invalid response value: ${response}`);
      }

      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      if (round.state !== "UPCOMING") {
        throw new Error("You can only join rounds that are upcoming");
      }

      const participants = [...round.participants];
      if (participants.find((p) => p.discordID === discordID)) {
        throw new Error("Participant already joined the round");
      }

      const participant: GraphQLParticipant = {
        discordID,
        response,
        tagNumber,
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
  ): Promise<GraphQLRound> {
    try {
      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      if (round.state !== "IN_PROGRESS") {
        throw new Error(
          "Scores can only be submitted for rounds that are in progress"
        );
      }

      const scores = [...round.scores];
      if (scores.find((s) => s.discordID === discordID)) {
        throw new Error("Score for this participant already exists");
      }

      scores.push({ discordID, score, tagNumber });

      await this.db
        .update(RoundModel)
        .set({ scores: JSON.stringify(scores) })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return { ...round, scores };
    } catch (error) {
      console.error("Error submitting score:", error);
      throw new Error("Failed to submit score");
    }
  }

  async finalizeAndProcessScores(
    roundID: string,
    scoreService: ScoreService
  ): Promise<GraphQLRound> {
    try {
      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      if (round.finalized) {
        throw new Error("Round has already been finalized");
      }

      await scoreService.processScores(roundID, round.scores);

      round.state = "FINALIZED";
      round.finalized = true;

      await this.db
        .update(RoundModel)
        .set({ state: "FINALIZED", finalized: true })
        .where(eq(RoundModel.roundID, Number(roundID)));

      return round;
    } catch (error) {
      console.error("Error finalizing and processing scores:", error);
      throw new Error("Failed to finalize and process scores");
    }
  }

  async editRound(
    roundID: string,
    input: EditRoundInput
  ): Promise<GraphQLRound> {
    try {
      const round = await this.getRound(roundID);
      if (!round) throw new Error("Round not found");

      const { roundID: _, ...updatedRoundData } = { ...round, ...input };

      await this.db
        .update(RoundModel)
        .set(updatedRoundData)
        .where(eq(RoundModel.roundID, Number(roundID)));

      return this.mapRoundToGraphQL(updatedRoundData);
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
  ): Promise<GraphQLRound> {
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

  private mapRoundToGraphQL(round: any): GraphQLRound {
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
      participants: safeParse(round.participants, []), // Ensure fallback
      scores: safeParse(round.scores, []), // Ensure fallback
      finalized: round.finalized,
      creatorID: round.creatorID,
      state: round.state as RoundState,
    };
  }
}
