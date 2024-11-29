// src/modules/api-gateway/request-handlers/validation.handler.ts
import { RequestHandler } from "./request-handler.interface";

export class ValidationHandler implements RequestHandler {
  private nextHandler: RequestHandler | null = null;

  setNext(handler: RequestHandler): this {
    this.nextHandler = handler;
    return this;
  }

  async handle(request: any): Promise<any> {
    // Implement validation logic here
    // ...

    if (this.nextHandler) {
      return this.nextHandler.handle(request);
    }

    return;
  }
}
