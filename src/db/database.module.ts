// src/db/database.module.ts
import { Module, Global } from "@nestjs/common";
import { db } from "src/database"; // Assuming you have a database.ts file

@Global() // Make it globally available
@Module({
  providers: [
    {
      provide: "DATABASE_CONNECTION",
      useValue: db,
    },
  ],
  exports: ["DATABASE_CONNECTION"],
})
export class DatabaseModule {}
