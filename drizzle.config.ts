import "dotenv/config";
import { defineConfig } from "drizzle-kit";

export default defineConfig({
  out: "./drizzle",
  schema: "./src/db/migrations/**",
  dialect: "postgresql",
  dbCredentials: {
    url: "postgres://postgres:mypassword@localhost:5432/test",
  },
});
