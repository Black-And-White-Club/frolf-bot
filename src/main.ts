import { NestFactory } from "@nestjs/core";
import { AppModule } from "./app.module";

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  // Optional: Enable CORS if needed
  app.enableCors();

  // Start the application and listen on the specified port
  const PORT = process.env.PORT || 4000;
  await app.listen(PORT);
  console.info(`Server is running on http://localhost:${PORT}/graphql`);
}

bootstrap();
