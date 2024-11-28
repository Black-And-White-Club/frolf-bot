import { GraphQLResolveInfo } from 'graphql';
export type Maybe<T> = T | null | undefined;
export type InputMaybe<T> = T | null | undefined;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
export type Omit<T, K extends keyof T> = Pick<T, Exclude<keyof T, K>>;
export type RequireFields<T, K extends keyof T> = Omit<T, K> & { [P in K]-?: NonNullable<T[P]> };
export type EnumResolverSignature<T, AllowedValues = any> = { [key in keyof T]?: AllowedValues };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string | number; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
};

export type CreateUserResponse = {
  __typename?: 'CreateUserResponse';
  error?: Maybe<Scalars['String']['output']>;
  success: Scalars['Boolean']['output'];
  user?: Maybe<User>;
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
  createUser: CreateUserResponse;
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
  updateUser: UpdateUserResponse;
};


export type MutationcreateUserArgs = {
  input: UserInput;
};


export type MutationdeleteRoundArgs = {
  roundID: Scalars['ID']['input'];
};


export type MutationeditRoundArgs = {
  input: EditRoundInput;
  roundID: Scalars['ID']['input'];
};


export type MutationfinalizeAndProcessScoresArgs = {
  roundID: Scalars['ID']['input'];
};


export type MutationjoinRoundArgs = {
  input: JoinRoundInput;
};


export type MutationlinkTagArgs = {
  discordID: Scalars['ID']['input'];
  newTagNumber: Scalars['Int']['input'];
};


export type MutationmanualTagUpdateArgs = {
  discordID: Scalars['ID']['input'];
  newTagNumber: Scalars['Int']['input'];
};


export type MutationprocessScoresArgs = {
  input: ProcessScoresInput;
};


export type MutationreceiveScoresArgs = {
  scores: Array<ScoreData>;
};


export type MutationscheduleRoundArgs = {
  input: ScheduleRoundInput;
};


export type MutationsubmitScoreArgs = {
  roundID: Scalars['ID']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};


export type MutationupdateParticipantResponseArgs = {
  response: Response;
  roundID: Scalars['ID']['input'];
};


export type MutationupdateScoreArgs = {
  discordID: Scalars['String']['input'];
  roundID: Scalars['ID']['input'];
  score: Scalars['Int']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};


export type MutationupdateTagArgs = {
  discordID: Scalars['ID']['input'];
  tagNumber: Scalars['Int']['input'];
};


export type MutationupdateUserArgs = {
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
  getScoresForRound: Array<Score>;
  getUser?: Maybe<User>;
  getUserScore: Score;
  getUserTag?: Maybe<TagNumber>;
};


export type QuerygetLeaderboardArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  page?: InputMaybe<Scalars['Int']['input']>;
};


export type QuerygetRoundArgs = {
  roundID: Scalars['ID']['input'];
};


export type QuerygetRoundsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};


export type QuerygetScoresForRoundArgs = {
  roundID: Scalars['String']['input'];
};


export type QuerygetUserArgs = {
  discordID: Scalars['String']['input'];
};


export type QuerygetUserScoreArgs = {
  discordID: Scalars['String']['input'];
  roundID: Scalars['String']['input'];
};


export type QuerygetUserTagArgs = {
  discordID: Scalars['ID']['input'];
};

export type Response =
  | 'ACCEPT'
  | 'DECLINE'
  | 'TENTATIVE';

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

export type RoundState =
  | 'DELETED'
  | 'FINALIZED'
  | 'IN_PROGRESS'
  | 'UPCOMING';

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
  discordID: Scalars['String']['output'];
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

export type UpdateUserResponse = {
  __typename?: 'UpdateUserResponse';
  error?: Maybe<Scalars['String']['output']>;
  success: Scalars['Boolean']['output'];
  user?: Maybe<User>;
};

export type User = {
  __typename?: 'User';
  createdAt: Scalars['String']['output'];
  deletedAt?: Maybe<Scalars['String']['output']>;
  discordID: Scalars['String']['output'];
  name: Scalars['String']['output'];
  role: UserRole;
  tagNumber?: Maybe<Scalars['Int']['output']>;
  updatedAt: Scalars['String']['output'];
};

export type UserInput = {
  discordID: Scalars['String']['input'];
  name: Scalars['String']['input'];
  tagNumber?: InputMaybe<Scalars['Int']['input']>;
};

export type UserRole =
  | 'ADMIN'
  | 'EDITOR'
  | 'RATTLER';



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
  CreateUserResponse: ResolverTypeWrapper<Omit<CreateUserResponse, 'user'> & { user?: Maybe<ResolversTypes['User']> }>;
  String: ResolverTypeWrapper<Scalars['String']['output']>;
  Boolean: ResolverTypeWrapper<Scalars['Boolean']['output']>;
  EditRoundInput: EditRoundInput;
  JoinRoundInput: JoinRoundInput;
  ID: ResolverTypeWrapper<Scalars['ID']['output']>;
  Leaderboard: ResolverTypeWrapper<Omit<Leaderboard, 'users'> & { users: Array<ResolversTypes['User']> }>;
  Mutation: ResolverTypeWrapper<{}>;
  Int: ResolverTypeWrapper<Scalars['Int']['output']>;
  Participant: ResolverTypeWrapper<Omit<Participant, 'response'> & { response: ResolversTypes['Response'] }>;
  ProcessScoresInput: ProcessScoresInput;
  Query: ResolverTypeWrapper<{}>;
  Response: ResolverTypeWrapper<'ACCEPT' | 'TENTATIVE' | 'DECLINE'>;
  Round: ResolverTypeWrapper<Omit<Round, 'participants' | 'state'> & { participants: Array<ResolversTypes['Participant']>, state: ResolversTypes['RoundState'] }>;
  RoundState: ResolverTypeWrapper<'UPCOMING' | 'IN_PROGRESS' | 'FINALIZED' | 'DELETED'>;
  ScheduleRoundInput: ScheduleRoundInput;
  Score: ResolverTypeWrapper<Score>;
  ScoreData: ScoreData;
  ScoreInput: ScoreInput;
  TagNumber: ResolverTypeWrapper<TagNumber>;
  UpdateUserInput: UpdateUserInput;
  UpdateUserResponse: ResolverTypeWrapper<Omit<UpdateUserResponse, 'user'> & { user?: Maybe<ResolversTypes['User']> }>;
  User: ResolverTypeWrapper<Omit<User, 'role'> & { role: ResolversTypes['UserRole'] }>;
  UserInput: UserInput;
  UserRole: ResolverTypeWrapper<'ADMIN' | 'EDITOR' | 'RATTLER'>;
};

/** Mapping between all available schema types and the resolvers parents */
export type ResolversParentTypes = {
  CreateUserResponse: Omit<CreateUserResponse, 'user'> & { user?: Maybe<ResolversParentTypes['User']> };
  String: Scalars['String']['output'];
  Boolean: Scalars['Boolean']['output'];
  EditRoundInput: EditRoundInput;
  JoinRoundInput: JoinRoundInput;
  ID: Scalars['ID']['output'];
  Leaderboard: Omit<Leaderboard, 'users'> & { users: Array<ResolversParentTypes['User']> };
  Mutation: {};
  Int: Scalars['Int']['output'];
  Participant: Participant;
  ProcessScoresInput: ProcessScoresInput;
  Query: {};
  Round: Omit<Round, 'participants'> & { participants: Array<ResolversParentTypes['Participant']> };
  ScheduleRoundInput: ScheduleRoundInput;
  Score: Score;
  ScoreData: ScoreData;
  ScoreInput: ScoreInput;
  TagNumber: TagNumber;
  UpdateUserInput: UpdateUserInput;
  UpdateUserResponse: Omit<UpdateUserResponse, 'user'> & { user?: Maybe<ResolversParentTypes['User']> };
  User: User;
  UserInput: UserInput;
};

export type CreateUserResponseResolvers<ContextType = any, ParentType extends ResolversParentTypes['CreateUserResponse'] = ResolversParentTypes['CreateUserResponse']> = {
  error?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  success?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type LeaderboardResolvers<ContextType = any, ParentType extends ResolversParentTypes['Leaderboard'] = ResolversParentTypes['Leaderboard']> = {
  placements?: Resolver<Array<ResolversTypes['TagNumber']>, ParentType, ContextType>;
  users?: Resolver<Array<ResolversTypes['User']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type MutationResolvers<ContextType = any, ParentType extends ResolversParentTypes['Mutation'] = ResolversParentTypes['Mutation']> = {
  createUser?: Resolver<ResolversTypes['CreateUserResponse'], ParentType, ContextType, RequireFields<MutationcreateUserArgs, 'input'>>;
  deleteRound?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType, RequireFields<MutationdeleteRoundArgs, 'roundID'>>;
  editRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationeditRoundArgs, 'input' | 'roundID'>>;
  finalizeAndProcessScores?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationfinalizeAndProcessScoresArgs, 'roundID'>>;
  joinRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationjoinRoundArgs, 'input'>>;
  linkTag?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationlinkTagArgs, 'discordID' | 'newTagNumber'>>;
  manualTagUpdate?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationmanualTagUpdateArgs, 'discordID' | 'newTagNumber'>>;
  processScores?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, RequireFields<MutationprocessScoresArgs, 'input'>>;
  receiveScores?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, RequireFields<MutationreceiveScoresArgs, 'scores'>>;
  scheduleRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationscheduleRoundArgs, 'input'>>;
  submitScore?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationsubmitScoreArgs, 'roundID' | 'score'>>;
  updateParticipantResponse?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<MutationupdateParticipantResponseArgs, 'response' | 'roundID'>>;
  updateScore?: Resolver<ResolversTypes['Score'], ParentType, ContextType, RequireFields<MutationupdateScoreArgs, 'discordID' | 'roundID' | 'score'>>;
  updateTag?: Resolver<ResolversTypes['TagNumber'], ParentType, ContextType, RequireFields<MutationupdateTagArgs, 'discordID' | 'tagNumber'>>;
  updateUser?: Resolver<ResolversTypes['UpdateUserResponse'], ParentType, ContextType, RequireFields<MutationupdateUserArgs, 'input'>>;
};

export type ParticipantResolvers<ContextType = any, ParentType extends ResolversParentTypes['Participant'] = ResolversParentTypes['Participant']> = {
  discordID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  response?: Resolver<ResolversTypes['Response'], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type QueryResolvers<ContextType = any, ParentType extends ResolversParentTypes['Query'] = ResolversParentTypes['Query']> = {
  getLeaderboard?: Resolver<ResolversTypes['Leaderboard'], ParentType, ContextType, RequireFields<QuerygetLeaderboardArgs, 'limit' | 'page'>>;
  getRound?: Resolver<ResolversTypes['Round'], ParentType, ContextType, RequireFields<QuerygetRoundArgs, 'roundID'>>;
  getRounds?: Resolver<Array<ResolversTypes['Round']>, ParentType, ContextType, Partial<QuerygetRoundsArgs>>;
  getScoresForRound?: Resolver<Array<ResolversTypes['Score']>, ParentType, ContextType, RequireFields<QuerygetScoresForRoundArgs, 'roundID'>>;
  getUser?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType, RequireFields<QuerygetUserArgs, 'discordID'>>;
  getUserScore?: Resolver<ResolversTypes['Score'], ParentType, ContextType, RequireFields<QuerygetUserScoreArgs, 'discordID' | 'roundID'>>;
  getUserTag?: Resolver<Maybe<ResolversTypes['TagNumber']>, ParentType, ContextType, RequireFields<QuerygetUserTagArgs, 'discordID'>>;
};

export type ResponseResolvers = EnumResolverSignature<{ ACCEPT?: any, DECLINE?: any, TENTATIVE?: any }, ResolversTypes['Response']>;

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

export type RoundStateResolvers = EnumResolverSignature<{ DELETED?: any, FINALIZED?: any, IN_PROGRESS?: any, UPCOMING?: any }, ResolversTypes['RoundState']>;

export type ScoreResolvers<ContextType = any, ParentType extends ResolversParentTypes['Score'] = ResolversParentTypes['Score']> = {
  discordID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
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

export type UpdateUserResponseResolvers<ContextType = any, ParentType extends ResolversParentTypes['UpdateUserResponse'] = ResolversParentTypes['UpdateUserResponse']> = {
  error?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  success?: Resolver<ResolversTypes['Boolean'], ParentType, ContextType>;
  user?: Resolver<Maybe<ResolversTypes['User']>, ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserResolvers<ContextType = any, ParentType extends ResolversParentTypes['User'] = ResolversParentTypes['User']> = {
  createdAt?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  deletedAt?: Resolver<Maybe<ResolversTypes['String']>, ParentType, ContextType>;
  discordID?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  name?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  role?: Resolver<ResolversTypes['UserRole'], ParentType, ContextType>;
  tagNumber?: Resolver<Maybe<ResolversTypes['Int']>, ParentType, ContextType>;
  updatedAt?: Resolver<ResolversTypes['String'], ParentType, ContextType>;
  __isTypeOf?: IsTypeOfResolverFn<ParentType, ContextType>;
};

export type UserRoleResolvers = EnumResolverSignature<{ ADMIN?: any, EDITOR?: any, RATTLER?: any }, ResolversTypes['UserRole']>;

export type Resolvers<ContextType = any> = {
  CreateUserResponse?: CreateUserResponseResolvers<ContextType>;
  Leaderboard?: LeaderboardResolvers<ContextType>;
  Mutation?: MutationResolvers<ContextType>;
  Participant?: ParticipantResolvers<ContextType>;
  Query?: QueryResolvers<ContextType>;
  Response?: ResponseResolvers;
  Round?: RoundResolvers<ContextType>;
  RoundState?: RoundStateResolvers;
  Score?: ScoreResolvers<ContextType>;
  TagNumber?: TagNumberResolvers<ContextType>;
  UpdateUserResponse?: UpdateUserResponseResolvers<ContextType>;
  User?: UserResolvers<ContextType>;
  UserRole?: UserRoleResolvers;
};

