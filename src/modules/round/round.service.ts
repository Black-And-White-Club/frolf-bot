// round.service.ts
import { Inject, Injectable } from "@nestjs/common";
import { drizzle } from "drizzle-orm/node-postgres";
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

@Injectable()
export class RoundService {
  constructor(
    @Inject("DATABASE_CONNECTION")
    private readonly db: ReturnType<typeof drizzle>
  ) {
    console.log("RoundService db:", this.db);
  }

  async getRounds(
    limit: number = 10,
    offset: number = 0
  ): Promise<GraphQLRound[]> {
    const rounds = await this.db
      .select()
      .from(RoundModel)
      .limit(limit)
      .offset(offset);

    return rounds.map((round) => this.mapRoundToGraphQL(round));
  }

  async getRound(roundID: string): Promise<GraphQLRound | null> {
    const rounds = await this.db
      .select()
      .from(RoundModel)
      .where(eq(RoundModel.roundID, Number(roundID)));

    if (rounds.length > 0) {
      return this.mapRoundToGraphQL(rounds[0]);
    }
    return null;
  }

  async scheduleRound(input: ScheduleRoundInput): Promise<GraphQLRound> {
    try {
      // Add try...catch block here
      const errors = await validate(input);
      if (errors.length > 0) {
        throw new Error("Validation failed!");
      }

      const roundData = {
        title: input.title,
        location: input.location,
        eventType: input.eventType || null,
        date: input.date,
        time: input.time,
        participants: JSON.stringify([]),
        scores: JSON.stringify([]),
        finalized: false,
        creatorID: input.creatorID,
        state: "UPCOMING",
      };

      const [round] = await this.db
        .insert(RoundModel)
        .values(roundData)
        .returning();

      return this.mapRoundToGraphQL(round);
    } catch (error) {
      // Handle the error appropriately (e.g., log it, re-throw it)
      console.error("Error during validation or database operation:", error);
      throw error;
    }
  }

  async joinRound(
    input: JoinRoundInput & { tagNumber: number | null }
  ): Promise<GraphQLRound> {
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

    const participant: GraphQLParticipant = { discordID, response, tagNumber };
    participants.push(participant);

    await this.db
      .update(RoundModel)
      .set({ participants: JSON.stringify(participants) })
      .where(eq(RoundModel.roundID, Number(roundID)));

    return { ...round, participants };
  }

  async submitScore(
    roundID: string,
    discordID: string,
    score: number,
    tagNumber: number | null
  ): Promise<GraphQLRound> {
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
  }

  async finalizeAndProcessScores(
    roundID: string,
    scoreService: ScoreService
  ): Promise<GraphQLRound> {
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
  }

  async editRound(
    roundID: string,
    input: EditRoundInput
  ): Promise<GraphQLRound> {
    const round = await this.getRound(roundID);
    if (!round) throw new Error("Round not found");

    const { roundID: _, ...updatedRoundData } = { ...round, ...input };

    await this.db
      .update(RoundModel)
      .set(updatedRoundData)
      .where(eq(RoundModel.roundID, Number(roundID)));

    return this.mapRoundToGraphQL(updatedRoundData);
  }

  async deleteRound(roundID: string, userID: string): Promise<boolean> {
    const round = await this.getRound(roundID);
    if (!round) throw new Error("Round not found");
    if (round.creatorID !== userID)
      throw new Error("Only the creator can delete the round");

    await this.db
      .delete(RoundModel)
      .where(eq(RoundModel.roundID, Number(roundID)));
    return true;
  }

  async updateParticipantResponse(
    roundID: string,
    discordID: string,
    response: Response
  ): Promise<GraphQLRound> {
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
  }

  private mapRoundToGraphQL(round: any): GraphQLRound {
    return {
      __typename: "Round",
      roundID: round.roundID,
      title: round.title,
      location: round.location,
      eventType: round.eventType,
      date: round.date,
      time: round.time,
      participants: round.participants || [],
      scores: round.scores || [],
      finalized: round.finalized || false,
      creatorID: round.creatorID,
      state: round.state as RoundState,
    };
  }
}
