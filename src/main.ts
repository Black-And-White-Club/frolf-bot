import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";
import * as dotenv from "dotenv";
import { Logger, ValidationPipe } from "@nestjs/common"; // Import Logger and ValidationPipe

async function bootstrap() {
  dotenv.config();

  const app = await NestFactory.create(AppModule);

  // Use a global validation pipe (optional but recommended)
  app.useGlobalPipes(new ValidationPipe());

  const PORT = process.env.PORT || 4000;
  await app.listen(PORT);
  Logger.log(`🚀 Server is running on http://localhost:${PORT}/graphql`); // Use Logger
}

bootstrap().catch((err) => {
  console.error("Error during application bootstrap:", err);
});
