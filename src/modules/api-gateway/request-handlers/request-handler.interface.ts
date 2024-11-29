// src/modules/api-gateway/request-handlers/request-handler.interface.ts

export interface RequestHandler {
  setNext(handler: RequestHandler): this;
  handle(request: any): Promise<any>;
}
