/* This file was automatically generated. DO NOT UPDATE MANUALLY. */
    import type   { Resolvers } from './types.generated';
    import    { getLeaderboard as Query_getLeaderboard } from './leaderboard/resolvers/Query/getLeaderboard';
import    { getRounds as Query_getRounds } from './round/resolvers/Query/getRounds';
import    { getUser as Query_getUser } from './user/resolvers/Query/getUser';
import    { getUserScore as Query_getUserScore } from './score/resolvers/Query/getUserScore';
import    { createUser as Mutation_createUser } from './user/resolvers/Mutation/createUser';
import    { deleteRound as Mutation_deleteRound } from './round/resolvers/Mutation/deleteRound';
import    { editRound as Mutation_editRound } from './round/resolvers/Mutation/editRound';
import    { finalizeRound as Mutation_finalizeRound } from './round/resolvers/Mutation/finalizeRound';
import    { joinRound as Mutation_joinRound } from './round/resolvers/Mutation/joinRound';
import    { scheduleRound as Mutation_scheduleRound } from './round/resolvers/Mutation/scheduleRound';
import    { submitScore as Mutation_submitScore } from './round/resolvers/Mutation/submitScore';
import    { EditLog } from './shared/resolvers/EditLog';
import    { Leaderboard } from './leaderboard/resolvers/Leaderboard';
import    { Participant } from './round/resolvers/Participant';
import    { Round } from './round/resolvers/Round';
import    { Score } from './score/resolvers/Score';
import    { Tag } from './leaderboard/resolvers/Tag';
import    { User } from './shared/resolvers/User';
    export const resolvers: Resolvers = {
      Query: { getLeaderboard: Query_getLeaderboard,getRounds: Query_getRounds,getUser: Query_getUser,getUserScore: Query_getUserScore },
      Mutation: { createUser: Mutation_createUser,deleteRound: Mutation_deleteRound,editRound: Mutation_editRound,finalizeRound: Mutation_finalizeRound,joinRound: Mutation_joinRound,scheduleRound: Mutation_scheduleRound,submitScore: Mutation_submitScore },
      
      EditLog: EditLog,
Leaderboard: Leaderboard,
Participant: Participant,
Round: Round,
Score: Score,
Tag: Tag,
User: User
    }