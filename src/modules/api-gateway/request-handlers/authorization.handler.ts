// src/modules/api-gateway/request-handlers/authorization.handler.ts
import { RequestHandler } from "./request-handler.interface";

export class AuthorizationHandler implements RequestHandler {
  private nextHandler: RequestHandler | null = null;

  setNext(handler: RequestHandler): this {
    this.nextHandler = handler;
    return this;
  }

  async handle(request: any): Promise<any> {
    // Implement authorization logic here (if needed)
    // ...

    if (this.nextHandler) {
      return this.nextHandler.handle(request);
    }

    return;
  }
}
