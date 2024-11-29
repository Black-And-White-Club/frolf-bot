// // src/modules/api-gateway/api-gateway.module.ts
// import { Module } from "@nestjs/common";
// import { GraphQLModule } from "@nestjs/graphql";
// import {
//   YogaGatewayDriver,
//   YogaGatewayDriverConfig,
// } from "@graphql-yoga/nestjs-federation";
// import {
//   LeaderboardModule,
//   RoundModule,
//   UserModule,
//   ScoreModule,
// } from "src/modules";
// import { ApiGatewayService } from "./api-gateway.service";
// import { readFileSync } from "fs";
// import { join } from "path";
// import { GraphQLContextProvider } from "../../context/graphql-context.provider";

// @Module({
//   imports: [
//     LeaderboardModule,
//     UserModule,
//     RoundModule,
//     ScoreModule,
//     GraphQLModule.forRootAsync<YogaGatewayDriverConfig>({
//       driver: YogaGatewayDriver,
//       useFactory: async () => ({
//         typePaths: ["./src/**/*.graphql"],
//         definitions: {
//           path: join(process.cwd(), "src/types.generated.ts"),
//           outputAs: "class",
//         },
//         gateway: {
//           supergraphSdl: readFileSync("./supergraph.graphql").toString(),
//           serviceList: [
//             { name: "user", url: "http://localhost:4000/v1/user" },
//             {
//               name: "leaderboard",
//               url: "http://localhost:4000/v1/leaderboard",
//             },
//             { name: "round", url: "http://localhost:4000/v1/round" },
//             { name: "score", url: "http://localhost:4000/v1/score" },
//           ],
//         },
//         server: {
//           cors: true,
//           path: "/v1/gateway",
//           context: ({ req }: any) => {
//             return { req };
//           },
//           playground: {
//             endpoint: "/v1/gateway", // Add the leading slash
//           },
//         },
//       }),
//     }),
//   ],
//   providers: [
//     ApiGatewayService,
//     {
//       provide: "API_GATEWAY_SERVICE",
//       useClass: ApiGatewayService,
//     },
//     GraphQLContextProvider,
//   ],
// })
// export class ApiGatewayModule {
//   constructor(private readonly apiGatewayService: ApiGatewayService) {}
// }
