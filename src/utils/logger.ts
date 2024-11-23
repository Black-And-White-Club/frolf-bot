// src/utils/logger.ts
import { Injectable } from "@nestjs/common";
import * as winston from "winston";

@Injectable()
export class LoggingService {
  private readonly logger: winston.Logger;

  constructor() {
    this.logger = winston.createLogger({
      level: "info",
      format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json()
      ),
      transports: [
        new winston.transports.Console(),
        new winston.transports.File({ filename: "error.log", level: "error" }),
      ],
    });
  }

  logError(message: string, meta?: any) {
    this.logger.error(message, meta);
  }

  logInfo(message: string) {
    this.logger.info(message);
  }

  logWarn(message: string) {
    this.logger.warn(message);
  }
}
