import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";
import * as dotenv from "dotenv";

async function bootstrap() {
  dotenv.config();

  const app = await NestFactory.create(AppModule);

  const PORT = process.env.PORT || 4000;
  await app.listen(PORT);
  console.log(`Server is running on http://localhost:4000/graphql`);
}

bootstrap().catch((err) => {
  console.error("Error during application bootstrap:", err);
});
