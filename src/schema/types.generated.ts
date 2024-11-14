import { GraphQLResolveInfo } from "graphql";
export type Maybe<T> = T | null | undefined;
export type InputMaybe<T> = T | null | undefined;
export type Exact<T extends { [key: string]: unknown }> = {
  [K in keyof T]: T[K];
};
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]?: Maybe<T[SubKey]>;
};
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & {
  [SubKey in K]: Maybe<T[SubKey]>;
};
export type MakeEmpty<
  T extends { [key: string]: unknown },
  K extends keyof T
> = { [_ in K]?: never };
export type Incremental<T> =
  | T
  | {
      [P in keyof T]?: P extends " $fragmentName" | "__typename" ? T[P] : never;
    };
export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type RequireFields<T, K extends keyof T> = Omit<T, K> & {
  [P in K]-?: NonNullable<T[P]>;
};
export type EnumResolverSignature<T, AllowedValues = any> = {
  [key in keyof T]?: AllowedValues;
};
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string | number };
  String: { input: string; output: string };
  Boolean: { input: boolean; output: boolean };
  Int: { input: number; output: number };
  Float: { input: number; output: number };
};

export type EditLog = {
  __typename?: "EditLog";
  changes: Scalars["String"]["output"];
  editorID: Scalars["String"]["output"];
  timestamp: Scalars["String"]["output"];
};

export type JoinRoundInput = {
  response: Response;
  roundID: Scalars["ID"]["input"];
  userID: Scalars["ID"]["input"];
};

export type Leaderboard = {
  __typename?: "Leaderboard";
  editLog: Array<EditLog>;
  placements: Array<Tag>;
  users: Array<User>;
};

export type Mutation = {
  __typename?: "Mutation";
  createUser: User;
  deleteRound: Scalars["Boolean"]["output"];
  editRound: Round;
  finalizeRound: Round;
  joinRound: Round;
  scheduleRound: Round;
  submitScore: Round;
};

export type MutationcreateUserArgs = {
  input: UserInput;
};

export type MutationdeleteRoundArgs = {
  roundID: Scalars["ID"]["input"];
};

export type MutationeditRoundArgs = {
  input: RoundInput;
  roundID: Scalars["ID"]["input"];
};

export type MutationfinalizeRoundArgs = {
  roundID: Scalars["ID"]["input"];
};

export type MutationjoinRoundArgs = {
  input: JoinRoundInput;
};

export type MutationscheduleRoundArgs = {
  input: RoundInput;
};

export type MutationsubmitScoreArgs = {
  roundID: Scalars["ID"]["input"];
  score: Scalars["Int"]["input"];
  userID: Scalars["ID"]["input"];
};

export type Participant = {
  __typename?: "Participant";
  rank: Scalars["Int"]["output"];
  response: Response;
  user: User;
};

export type Query = {
  __typename?: "Query";
  getLeaderboard?: Maybe<Leaderboard>;
  getRounds: Array<Round>;
  getUser?: Maybe<User>;
  getUserScore?: Maybe<Scalars["Int"]["output"]>;
};

export type QuerygetRoundsArgs = {
  limit?: InputMaybe<Scalars["Int"]["input"]>;
  offset?: InputMaybe<Scalars["Int"]["input"]>;
};

export type QuerygetUserArgs = {
  discordID: Scalars["String"]["input"];
};

export type QuerygetUserScoreArgs = {
  userID: Scalars["String"]["input"];
};

export type Response = "ACCEPT" | "DECLINE" | "TENTATIVE";

export type Round = {
  __typename?: "Round";
  creatorID: Scalars["String"]["output"];
  date: Scalars["String"]["output"];
  editLog: Array<EditLog>;
  eventType?: Maybe<Scalars["String"]["output"]>;
  finalized: Scalars["Boolean"]["output"];
  id: Scalars["ID"]["output"];
  location: Scalars["String"]["output"];
  participants: Array<Participant>;
  scores: Array<Score>;
  time: Scalars["String"]["output"];
  title: Scalars["String"]["output"];
};

export type RoundInput = {
  date: Scalars["String"]["input"];
  eventType?: InputMaybe<Scalars["String"]["input"]>;
  location: Scalars["String"]["input"];
  time: Scalars["String"]["input"];
  title: Scalars["String"]["input"];
};

export type Score = {
  __typename?: "Score";
  editLog: Array<EditLog>;
  score: Scalars["Int"]["output"];
  userID: Scalars["ID"]["output"];
};

export type Tag = {
  __typename?: "Tag";
  durationHeld?: Maybe<Scalars["Int"]["output"]>;
  editLog: Array<EditLog>;
  id: Scalars["ID"]["output"];
  lastPlayed: Scalars["String"]["output"];
  name: Scalars["String"]["output"];
  tagNumber?: Maybe<Scalars["Int"]["output"]>;
};

export type User = {
  __typename?: "User";
  discordID: Scalars["String"]["output"];
  editLog: Array<EditLog>;
  id: Scalars["ID"]["output"];
  name: Scalars["String"]["output"];
  role: Scalars["String"]["output"];
  rounds: Array<Round>;
  tagNumber?: Maybe<Scalars["Int"]["output"]>;
};

export type UserInput = {
  discordID: Scalars["String"]["input"];
  name: Scalars["String"]["input"];
};

export type ResolverTypeWrapper<T> = Promise<T> | T;

export type ResolverWithResolve<TResult, TParent, TContext, TArgs> = {
  resolve: ResolverFn<TResult, TParent, TContext, TArgs>;
};
export type Resolver<TResult, TParent = {}, TContext = {}, TArgs = {}> =
  | ResolverFn<TResult, TParent, TContext, TArgs>
  | ResolverWithResolve<TResult, TParent, TContext, TArgs>;

export type ResolverFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => Promise<TResult> | TResult;

export type SubscriptionSubscribeFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => AsyncIterable<TResult> | Promise<AsyncIterable<TResult>>;

export type SubscriptionResolveFn<TResult, TParent, TContext, TArgs> = (
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => TResult | Promise<TResult>;

export interface SubscriptionSubscriberObject<
  TResult,
  TKey extends string,
  TParent,
  TContext,
  TArgs
> {
  subscribe: SubscriptionSubscribeFn<
    { [key in TKey]: TResult },
    TParent,
    TContext,
    TArgs
  >;
  resolve?: SubscriptionResolveFn<
    TResult,
    { [key in TKey]: TResult },
    TContext,
    TArgs
  >;
}

export interface SubscriptionResolverObject<TResult, TParent, TContext, TArgs> {
  subscribe: SubscriptionSubscribeFn<any, TParent, TContext, TArgs>;
  resolve: SubscriptionResolveFn<TResult, any, TContext, TArgs>;
}

export type SubscriptionObject<
  TResult,
  TKey extends string,
  TParent,
  TContext,
  TArgs
> =
  | SubscriptionSubscriberObject<TResult, TKey, TParent, TContext, TArgs>
  | SubscriptionResolverObject<TResult, TParent, TContext, TArgs>;

export type SubscriptionResolver<
  TResult,
  TKey extends string,
  TParent = {},
  TContext = {},
  TArgs = {}
> =
  | ((
      ...args: any[]
    ) => SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>)
  | SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>;

export type TypeResolveFn<TTypes, TParent = {}, TContext = {}> = (
  parent: TParent,
  context: TContext,
  info: GraphQLResolveInfo
) => Maybe<TTypes> | Promise<Maybe<TTypes>>;

export type IsTypeOfResolverFn<T = {}, TContext = {}> = (
  obj: T,
  context: TContext,
  info: GraphQLResolveInfo
) => boolean | Promise<boolean>;

export type NextResolverFn<T> = () => Promise<T>;

export type DirectiveResolverFn<
  TResult = {},
  TParent = {},
  TContext = {},
  TArgs = {}
> = (
  next: NextResolverFn<TResult>,
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => TResult | Promise<TResult>;

/** Mapping between all available schema types and the resolvers types */
export type ResolversTypes = {
  EditLog: ResolverTypeWrapper<EditLog>;
  String: ResolverTypeWrapper<Scalars["String"]["output"]>;
  JoinRoundInput: JoinRoundInput;
  ID: ResolverTypeWrapper<Scalars["ID"]["output"]>;
  Leaderboard: ResolverTypeWrapper<
    Omit<Leaderboard, "users"> & { users: Array<ResolversTypes["User"]> }
  >;
  Mutation: ResolverTypeWrapper<{}>;
  Boolean: ResolverTypeWrapper<Scalars["Boolean"]["output"]>;
  Int: ResolverTypeWrapper<Scalars["Int"]["output"]>;
  Participant: ResolverTypeWrapper<
    Omit<Participant, "response" | "user"> & {
      response: ResolversTypes["Response"];
      user: ResolversTypes["User"];
    }
  >;
  Query: ResolverTypeWrapper<{}>;
  Response: ResolverTypeWrapper<"ACCEPT" | "TENTATIVE" | "DECLINE">;
  Round: ResolverTypeWrapper<
    Omit<Round, "participants"> & {
      participants: Array<ResolversTypes["Participant"]>;
    }
  >;
  RoundInput: RoundInput;
  Score: ResolverTypeWrapper<Score>;
  Tag: ResolverTypeWrapper<Tag>;
  User: ResolverTypeWrapper<
    Omit<User, "rounds"> & { rounds: Array<ResolversTypes["Round"]> }
  >;
  UserInput: UserInput;
};

/** Mapping between all available schema types and the resolvers parents */
export type ResolversParentTypes = {
  EditLog: EditLog;
  String: Scalars["String"]["output"];
  JoinRoundInput: JoinRoundInput;
  ID: Scalars["ID"]["output"];
  Leaderboard: Omit<Leaderboard, "users"> & {
    users: Array<ResolversParentTypes["User"]>;
  };
  Mutation: {};
  Boolean: Scalars["Boolean"]["output"];
  Int: Scalars["Int"]["output"];
  Participant: Omit<Participant, "user"> & {
    user: ResolversParentTypes["User"];
  };
  Query: {};
  Round: Omit<Round, "participants"> & {
    participants: Array<ResolversParentTypes["Participant"]>;
  };
  RoundInput: RoundInput;
  Score: Score;
  Tag: Tag;
  User: Omit<User, "rounds"> & { rounds: Array<ResolversParentTypes["Round"]> };
  UserInput: UserInput;
};

export type EditLogResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["EditLog"] = ResolversParentTypes["EditLog"]
> = {
  changes?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  editorID?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  timestamp?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LeaderboardResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Leaderboard"] = ResolversParentTypes["Leaderboard"]
> = {
  editLog?: Resolver<Array<ResolversTypes["EditLog"]>, ParentType, ContextType>;
  placements?: Resolver<Array<ResolversTypes["Tag"]>, ParentType, ContextType>;
  users?: Resolver<Array<ResolversTypes["User"]>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MutationResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Mutation"] = ResolversParentTypes["Mutation"]
> = {
  createUser?: Resolver<
    ResolversTypes["User"],
    ParentType,
    ContextType,
    RequireFields<MutationcreateUserArgs, "input">
  >;
  deleteRound?: Resolver<
    ResolversTypes["Boolean"],
    ParentType,
    ContextType,
    RequireFields<MutationdeleteRoundArgs, "roundID">
  >;
  editRound?: Resolver<
    ResolversTypes["Round"],
    ParentType,
    ContextType,
    RequireFields<MutationeditRoundArgs, "input" | "roundID">
  >;
  finalizeRound?: Resolver<
    ResolversTypes["Round"],
    ParentType,
    ContextType,
    RequireFields<MutationfinalizeRoundArgs, "roundID">
  >;
  joinRound?: Resolver<
    ResolversTypes["Round"],
    ParentType,
    ContextType,
    RequireFields<MutationjoinRoundArgs, "input">
  >;
  scheduleRound?: Resolver<
    ResolversTypes["Round"],
    ParentType,
    ContextType,
    RequireFields<MutationscheduleRoundArgs, "input">
  >;
  submitScore?: Resolver<
    ResolversTypes["Round"],
    ParentType,
    ContextType,
    RequireFields<MutationsubmitScoreArgs, "roundID" | "score" | "userID">
  >;
};

export type ParticipantResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Participant"] = ResolversParentTypes["Participant"]
> = {
  rank?: Resolver<ResolversTypes["Int"], ParentType, ContextType>;
  response?: Resolver<ResolversTypes["Response"], ParentType, ContextType>;
  user?: Resolver<ResolversTypes["User"], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QueryResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Query"] = ResolversParentTypes["Query"]
> = {
  getLeaderboard?: Resolver<
    Maybe<ResolversTypes["Leaderboard"]>,
    ParentType,
    ContextType
  >;
  getRounds?: Resolver<
    Array<ResolversTypes["Round"]>,
    ParentType,
    ContextType,
    Partial<QuerygetRoundsArgs>
  >;
  getUser?: Resolver<
    Maybe<ResolversTypes["User"]>,
    ParentType,
    ContextType,
    RequireFields<QuerygetUserArgs, "discordID">
  >;
  getUserScore?: Resolver<
    Maybe<ResolversTypes["Int"]>,
    ParentType,
    ContextType,
    RequireFields<QuerygetUserScoreArgs, "userID">
  >;
};

export type ResponseResolvers = EnumResolverSignature<
  { ACCEPT?: any; DECLINE?: any; TENTATIVE?: any },
  ResolversTypes["Response"]
>;

export type RoundResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Round"] = ResolversParentTypes["Round"]
> = {
  creatorID?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  date?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  editLog?: Resolver<Array<ResolversTypes["EditLog"]>, ParentType, ContextType>;
  eventType?: Resolver<
    Maybe<ResolversTypes["String"]>,
    ParentType,
    ContextType
  >;
  finalized?: Resolver<ResolversTypes["Boolean"], ParentType, ContextType>;
  id?: Resolver<ResolversTypes["ID"], ParentType, ContextType>;
  location?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  participants?: Resolver<
    Array<ResolversTypes["Participant"]>,
    ParentType,
    ContextType
  >;
  scores?: Resolver<Array<ResolversTypes["Score"]>, ParentType, ContextType>;
  time?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  title?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScoreResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Score"] = ResolversParentTypes["Score"]
> = {
  editLog?: Resolver<Array<ResolversTypes["EditLog"]>, ParentType, ContextType>;
  score?: Resolver<ResolversTypes["Int"], ParentType, ContextType>;
  userID?: Resolver<ResolversTypes["ID"], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["Tag"] = ResolversParentTypes["Tag"]
> = {
  durationHeld?: Resolver<
    Maybe<ResolversTypes["Int"]>,
    ParentType,
    ContextType
  >;
  editLog?: Resolver<Array<ResolversTypes["EditLog"]>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes["ID"], ParentType, ContextType>;
  lastPlayed?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  name?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes["Int"]>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserResolvers<
  ContextType = any,
  ParentType extends ResolversParentTypes["User"] = ResolversParentTypes["User"]
> = {
  discordID?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  editLog?: Resolver<Array<ResolversTypes["EditLog"]>, ParentType, ContextType>;
  id?: Resolver<ResolversTypes["ID"], ParentType, ContextType>;
  name?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  role?: Resolver<ResolversTypes["String"], ParentType, ContextType>;
  rounds?: Resolver<Array<ResolversTypes["Round"]>, ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes["Int"]>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type Resolvers<ContextType = any> = {
  EditLog?: EditLogResolvers<ContextType>;
  Leaderboard?: LeaderboardResolvers<ContextType>;
  Mutation?: MutationResolvers<ContextType>;
  Participant?: ParticipantResolvers<ContextType>;
  Query?: QueryResolvers<ContextType>;
  Response?: ResponseResolvers;
  Round?: RoundResolvers<ContextType>;
  Score?: ScoreResolvers<ContextType>;
  Tag?: TagResolvers<ContextType>;
  User?: UserResolvers<ContextType>;
};
