import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";
import * as dotenv from "dotenv";
import { Logger, ValidationPipe } from "@nestjs/common";

async function bootstrap() {
  dotenv.config();

  const app = await NestFactory.create(AppModule);

  // Enable CORS (Cross-Origin Resource Sharing)
  // app.enableCors();

  app.useGlobalPipes(new ValidationPipe());

  const PORT = process.env.PORT || 4000;
  await app.listen(PORT);
  Logger.log(`🚀 Server is running on http://localhost:${PORT}`);
}

bootstrap().catch((err) => {
  console.error("Error during application bootstrap:", err);
});
