// src/enums/index.ts
export enum UserRole {
  Rattler = "RATTLER",
  Admin = "ADMIN",
  Editor = "EDITOR",
}

export enum Response {
  Accept = "ACCEPT",
  Tentative = "TENTATIVE",
  Decline = "DECLINE",
}

export enum RoundState {
  Upcoming = "UPCOMING",
  InProgress = "IN_PROGRESS",
  Finalized = "FINALIZED",
}

export enum UpdateTagSource {
  ProcessScores = "processScores",
  Manual = "manual",
  CreateUser = "createUser",
}
