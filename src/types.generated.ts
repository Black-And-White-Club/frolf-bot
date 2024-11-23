import { GraphQLResolveInfo } from 'graphql';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
export type RequireFields<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
};

export type EditRoundInput = {
  date?: InputMaybe<Scalars['String']['input']>;
  eventType?: InputMaybe<Scalars['String']['input']>;
  location?: InputMaybe<Scalars['String']['input']>;
  time?: InputMaybe<Scalars['String']['input']>;
  title?: InputMaybe<Scalars['String']['input']>;
};

export type JoinRoundInput = {
  discordID: Scalars['String']['input'];
  response: Response;
  roundID: Scalars['ID']['input'];
};

export type Leaderboard = {
  __typename?: 'Leaderboard';
  placements: Array<TagNumber>;
  users: Array<User>;
};

export type Mutation = {
  __typename?: 'Mutation';
  createUser: User;
  deleteRound: Scalars['Boolean']['output'];
  editRound: Round;
  finalizeAndProcessScores: Round;
  joinRound: Round;
  linkTag: TagNumber;
  manualTagUpdate: TagNumber;
  processScores: Leaderboard;
  receiveScores: Leaderboard;
  scheduleRound: Round;
  submitScore: Round;
  updateParticipantResponse: Round;
  updateScore: Score;
  updateTag: TagNumber;
  updateUser: User;
};


export type MutationCreateUserArgs = {
  input: UserInput;
};


export type MutationDeleteRoundArgs = {
  roundID: Scalars['ID']['input'];
};


export type MutationEditRoundArgs = {
  input: EditRoundInput;
  roundID: Scalars['ID']['input'];
};


export type MutationFinalizeAndProcessScoresArgs = {
  roundID: Scalars['ID']['input'];
};


export type MutationJoinRoundArgs = {
  input: JoinRoundInput;
};


export type MutationLinkTagArgs = {
  discordID: Scalars['ID']['input'];
  newTagNumber: Scalars['Int']['input'];
};


export type MutationManualTagUpdateArgs = {
  discordID: Scalars['ID']['input'];
  newTagNumber: Scalars['Int']['input'];
};


export type MutationProcessScoresArgs = {
  input: ProcessScoresInput;
};


export type MutationReceiveScoresArgs = {
  scores: Array<ScoreData>;
};


export type MutationScheduleRoundArgs = {
  input: ScheduleRoundInput;
};


export type MutationSubmitScoreArgs = {
  roundID: Scalars['ID']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};


export type MutationUpdateParticipantResponseArgs = {
  response: Response;
  roundID: Scalars['ID']['input'];
};


export type MutationUpdateScoreArgs = {
  discordID: Scalars['String']['input'];
  roundID: Scalars['ID']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};


export type MutationUpdateTagArgs = {
  discordID: Scalars['ID']['input'];
  tagNumber: Scalars['Int']['input'];
};


export type MutationUpdateUserArgs = {
  input: UpdateUserInput;
};

export type Participant = {
  __typename?: 'Participant';
  discordID: Scalars['String']['output'];
  response: Response;
  tagNumber?: Maybe<Scalars['Int']['output']>;
};

export type ProcessScoresInput = {
  roundID: Scalars['ID']['input'];
  scores: Array<ScoreInput>;
};

export type Query = {
  __typename?: 'Query';
  getLeaderboard: Leaderboard;
  getRound: Round;
  getRounds: Array<Round>;
  getUser?: Maybe<User>;
  getUserTag?: Maybe<TagNumber>;
};


export type QueryGetLeaderboardArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryGetRoundArgs = {
  roundID: Scalars['ID']['input'];
};


export type QueryGetRoundsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QueryGetUserArgs = {
  discordID: Scalars['String']['input'];
};


export type QueryGetUserTagArgs = {
  discordID: Scalars['ID']['input'];
};

export enum Response {
  Accept = 'ACCEPT',
  Decline = 'DECLINE',
  Tentative = 'TENTATIVE'
}

export type Round = {
  __typename?: 'Round';
  creatorID: Scalars['String']['output'];
  date: Scalars['String']['output'];
  eventType?: Maybe<Scalars['String']['output']>;
  finalized: Scalars['Boolean']['output'];
  location: Scalars['String']['output'];
  participants: Array<Participant>;
  roundID: Scalars['ID']['output'];
  scores: Array<Score>;
  state: RoundState;
  time: Scalars['String']['output'];
  title: Scalars['String']['output'];
};

export enum RoundState {
  Deleted = 'DELETED',
  Finalized = 'FINALIZED',
  InProgress = 'IN_PROGRESS',
  Upcoming = 'UPCOMING'
}

export type ScheduleRoundInput = {
  creatorID: Scalars['String']['input'];
  date: Scalars['String']['input'];
  eventType?: InputMaybe<Scalars['String']['input']>;
  location: Scalars['String']['input'];
  time: Scalars['String']['input'];
  title: Scalars['String']['input'];
};

export type Score = {
  __typename?: 'Score';
  discordID: Scalars['ID']['output'];
  score: Scalars['Int']['output'];
  tagNumber?: Maybe<Scalars['Int']['output']>;
};

export type ScoreData = {
  discordID: Scalars['ID']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};

export type ScoreInput = {
  discordID: Scalars['String']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};

export type TagNumber = {
  __typename?: 'TagNumber';
  discordID: Scalars['ID']['output'];
  durationHeld: Scalars['Int']['output'];
  lastPlayed: Scalars['String']['output'];
  tagNumber: Scalars['Int']['output'];
};

export type UpdateUserInput = {
  discordID?: InputMaybe<Scalars['String']['input']>;
  name?: InputMaybe<Scalars['String']['input']>;
  role?: InputMaybe<UserRole>;
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};

export type User = {
  __typename?: 'User';
  discordID: Scalars['String']['output'];
  name: Scalars['String']['output'];
  role: UserRole;
  tagNumber?: Maybe<Scalars['Int']['output']>;
};

export type UserInput = {
  discordID: Scalars['String']['input'];
  name: Scalars['String']['input'];
  role: UserRole;
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};

export enum UserRole {
  Admin = 'ADMIN',
  Editor = 'EDITOR',
  Rattler = 'RATTLER'
}



export type ResolverTypeWrapper<T> = Promise<T> | T;


export type ResolverWithResolve<TResult, TParent, TContext, TArgs> = {
  resolve: ResolverFn<TResult, TParent, TContext, TArgs>;
};
export type Resolver<TResult, TParent = {}, TContext = {}, TArgs = {}> = ResolverFn<TResult, TParent, TContext, TArgs> | ResolverWithResolve<TResult, TParent, TContext, TArgs>;

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

export interface SubscriptionSubscriberObject<TResult, TKey extends string, TParent, TContext, TArgs> {
  subscribe: SubscriptionSubscribeFn<{ [key in TKey]: TResult }, TParent, TContext, TArgs>;
  resolve?: SubscriptionResolveFn<TResult, { [key in TKey]: TResult }, TContext, TArgs>;
}

export interface SubscriptionResolverObject<TResult, TParent, TContext, TArgs> {
  subscribe: SubscriptionSubscribeFn<any, TParent, TContext, TArgs>;
  resolve: SubscriptionResolveFn<TResult, any, TContext, TArgs>;
}

export type SubscriptionObject<TResult, TKey extends string, TParent, TContext, TArgs> =
  | SubscriptionSubscriberObject<TResult, TKey, TParent, TContext, TArgs>
  | SubscriptionResolverObject<TResult, TParent, TContext, TArgs>;

export type SubscriptionResolver<TResult, TKey extends string, TParent = {}, TContext = {}, TArgs = {}> =
  | ((...args: any[]) => SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>)
  | SubscriptionObject<TResult, TKey, TParent, TContext, TArgs>;

export type TypeResolveFn<TTypes, TParent = {}, TContext = {}> = (
  parent: TParent,
  context: TContext,
  info: GraphQLResolveInfo
) => Maybe<TTypes> | Promise<Maybe<TTypes>>;

export type IsTypeOfResolverFn<T = {}, TContext = {}> = (obj: T, context: TContext, info: GraphQLResolveInfo) => boolean | Promise<boolean>;

export type NextResolverFn<T> = () => Promise<T>;

export type DirectiveResolverFn<TResult = {}, TParent = {}, TContext = {}, TArgs = {}> = (
  next: NextResolverFn<TResult>,
  parent: TParent,
  args: TArgs,
  context: TContext,
  info: GraphQLResolveInfo
) => TResult | Promise<TResult>;



/** Mapping between all available schema types and the resolvers types */
export type ResolversTypes = {
  Boolean: ResolverTypeWrapper<Scalars['Boolean']['output']>;
  EditRoundInput: EditRoundInput;
  ID: ResolverTypeWrapper<Scalars['ID']['output']>;
  Int: ResolverTypeWrapper<Scalars['Int']['output']>;
  JoinRoundInput: JoinRoundInput;
  Leaderboard: ResolverTypeWrapper<Leaderboard>;
  Mutation: ResolverTypeWrapper<{}>;
  Participant: ResolverTypeWrapper<Participant>;
  ProcessScoresInput: ProcessScoresInput;
  Query: ResolverTypeWrapper<{}>;
  Response: Response;
  Round: ResolverTypeWrapper<Round>;
  RoundState: RoundState;
  ScheduleRoundInput: ScheduleRoundInput;
  Score: ResolverTypeWrapper<Score>;
  ScoreData: ScoreData;
  ScoreInput: ScoreInput;
  String: ResolverTypeWrapper<Scalars['String']['output']>;
  TagNumber: ResolverTypeWrapper<TagNumber>;
  UpdateUserInput: UpdateUserInput;
  User: ResolverTypeWrapper<User>;
  UserInput: UserInput;
  UserRole: UserRole;
};

/** Mapping between all available schema types and the resolvers parents */
export type ResolversParentTypes = {
  Boolean: Scalars['Boolean']['output'];
  EditRoundInput: EditRoundInput;
  ID: Scalars['ID']['output'];
  Int: Scalars['Int']['output'];
  JoinRoundInput: JoinRoundInput;
  Leaderboard: Leaderboard;
  Mutation: {};
  Participant: Participant;
  ProcessScoresInput: ProcessScoresInput;
  Query: {};
  Round: Round;
  ScheduleRoundInput: ScheduleRoundInput;
  Score: Score;
  ScoreData: ScoreData;
  ScoreInput: ScoreInput;
  String: Scalars['String']['output'];
  TagNumber: TagNumber;
  UpdateUserInput: UpdateUserInput;
  User: User;
  UserInput: UserInput;
};

export type LeaderboardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Leaderboard'] = ResolversParentTypes['Leaderboard']> = {
  placements?: Resolver<Array<ResolversTypes['TagNumber']>, ParentType, ContextType>;
  users?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MutationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Mutation'] = ResolversParentTypes['Mutation']> = {
  createUser?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationCreateUserArgs, 'input'>>;
  deleteRound?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType, RequireFields<MutationDeleteRoundArgs, 'roundID'>>;
  editRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationEditRoundArgs, 'input' | 'roundID'>>;
  finalizeAndProcessScores?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationFinalizeAndProcessScoresArgs, 'roundID'>>;
  joinRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationJoinRoundArgs, 'input'>>;
  linkTag?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationLinkTagArgs, 'discordID' | 'newTagNumber'>>;
  manualTagUpdate?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationManualTagUpdateArgs, 'discordID' | 'newTagNumber'>>;
  processScores?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, RequireFields<MutationProcessScoresArgs, 'input'>>;
  receiveScores?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, RequireFields<MutationReceiveScoresArgs, 'scores'>>;
  scheduleRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationScheduleRoundArgs, 'input'>>;
  submitScore?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationSubmitScoreArgs, 'roundID' | 'score'>>;
  updateParticipantResponse?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationUpdateParticipantResponseArgs, 'response' | 'roundID'>>;
  updateScore?: Resolver<ResolversTypes['Score'], ParentType, ContextType, RequireFields<MutationUpdateScoreArgs, 'discordID' | 'roundID' | 'score'>>;
  updateTag?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationUpdateTagArgs, 'discordID' | 'tagNumber'>>;
  updateUser?: Resolver<ResolversTypes['User'], ParentType, ContextType, RequireFields<MutationUpdateUserArgs, 'input'>>;
};

export type ParticipantResolvers<ContextType = any, ParentType extends ResolversParentTypes['Participant'] = ResolversParentTypes['Participant']> = {
  discordID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  response?: Resolver<ResolversTypes['Response'], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QueryResolvers<ContextType = any, ParentType extends ResolversParentTypes['Query'] = ResolversParentTypes['Query']> = {
  getLeaderboard?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, Partial<QueryGetLeaderboardArgs>>;
  getRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<QueryGetRoundArgs, 'roundID'>>;
  getRounds?: Resolver<Array<ResolversTypes['Round']>, ParentType, ContextType, Partial<QueryGetRoundsArgs>>;
  getUser?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType, RequireFields<QueryGetUserArgs, 'discordID'>>;
  getUserTag?: Resolver<Maybe<ResolversTypes['TagNumber']>, ParentType, ContextType, RequireFields<QueryGetUserTagArgs, 'discordID'>>;
};

export type RoundResolvers<ContextType = any, ParentType extends ResolversParentTypes['Round'] = ResolversParentTypes['Round']> = {
  creatorID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  date?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  eventType?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  finalized?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  location?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  participants?: Resolver<Array<ResolversTypes['Participant']>, ParentType, ContextType>;
  roundID?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  scores?: Resolver<Array<ResolversTypes['Score']>, ParentType, ContextType>;
  state?: Resolver<ResolversTypes['RoundState'], ParentType, ContextType>;
  time?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  title?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type ScoreResolvers<ContextType = any, ParentType extends ResolversParentTypes['Score'] = ResolversParentTypes['Score']> = {
  discordID?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  score?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type TagNumberResolvers<ContextType = any, ParentType extends ResolversParentTypes['TagNumber'] = ResolversParentTypes['TagNumber']> = {
  discordID?: Resolver<ResolversTypes['ID'], ParentType, ContextType>;
  durationHeld?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  lastPlayed?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  tagNumber?: Resolver<ResolversTypes['Int'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserResolvers<ContextType = any, ParentType extends ResolversParentTypes['User'] = ResolversParentTypes['User']> = {
  discordID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  role?: Resolver<ResolversTypes['UserRole'], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type Resolvers<ContextType = any> = {
  Leaderboard?: LeaderboardResolvers<ContextType>;
  Mutation?: MutationResolvers<ContextType>;
  Participant?: ParticipantResolvers<ContextType>;
  Query?: QueryResolvers<ContextType>;
  Round?: RoundResolvers<ContextType>;
  Score?: ScoreResolvers<ContextType>;
  TagNumber?: TagNumberResolvers<ContextType>;
  User?: UserResolvers<ContextType>;
};

