# Project README

## Overview

This project is a GraphQL-based application that manages rounds of events, scores, and user interactions. It uses PostgreSQL as the database and Drizzle ORM for database interactions. The application is structured into various services, resolvers, and tests to ensure functionality and maintainability.

## Table of Contents

1. [Technologies Used](#technologies-used)
2. [Folder Structure](#folder-structure)
3. [Key Components](#key-components)
   - [Services](#services)
   - [Resolvers](#resolvers)
   - [Database Models](#database-models)
4. [Running the Application](#running-the-application)
5. [Testing](#testing)
6. [Feedback](#feedback)

## Technologies Used

- **Node.js**: JavaScript runtime for building the application.
- **GraphQL**: API query language used for data fetching.
- **PostgreSQL**: Relational database used to store data.
- **Drizzle ORM**: ORM used for database interactions.
- **Vitest**: Testing framework used for unit and integration tests.

## Folder Structure

```bash
.
└── src
    ├── __tests__
    │   ├── db
    │   └── graphql
    ├── db
    │   ├── helpers
    │   └── migrations
    ├── dto
    │   ├── leaderboard
    │   ├── round
    │   ├── score
    │   └── user
    ├── enums
    ├── middleware
    └── modules
        ├── leaderboard
        ├── round
        ├── score
        └── user

```

## Key Components

### Services

- **User Service**: Manages user data, including creation, retrieval, and updates.
- **RoundService**: Handles round-related operations, such as scheduling, joining, and finalizing rounds.
- **ScoreService**: Manages scores for users in rounds, including processing and updating scores.
- **LeaderboardService**: Maintains the leaderboard, linking users to tags and managing their scores.

### Resolvers

- **User Resolver**: Handles GraphQL queries and mutations related to users.
- **RoundResolver**: Manages GraphQL interactions for rounds.
- **ScoreResolver**: Processes score-related GraphQL queries and mutations.
- **LeaderboardResolver**: Manages leaderboard queries and mutations.

### Database Models

- **User **: Represents users in the system, including their Discord ID and role.
- **Round**: Represents rounds of events, including participants and scores.
- **Score**: Represents scores associated with users in specific rounds.
- **Leaderboard**: Represents the leaderboard information for users and their scores.

## Running the Application

1. **Clone the Repository**:
   ```bash
   git clone <repository-url>
   cd <repository-name>
   ```

## Install Dependencies

Make sure you have Node.js installed. Run the following command to install the necessary packages:

```bash
pnpm install
```

## Set Up Environment Variables

Create a `.env` file in the root of the project and add your PostgreSQL database connection string:

```bash
DATABASE_URL=postgres://<username>:<password>@localhost:5432/<database_name>
```

## Start the PostgreSQL Database

```
Make sure your PostgreSQL database is running. You can use Docker or any local installation of PostgreSQL.

```

## Run the Application Locally

To run the application in development mode, use the following command:

```bash
pnpm dev

```

## Access the GraphQL API

Once the application is running, you can access the GraphQL API at:

```bash
http://localhost:4000/graphql
```

## Testing

To run the tests for the application, execute the following command:

```bash
pnpm test
```
