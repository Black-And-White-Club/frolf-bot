// src/middleware/auth.guard.ts
import { CanActivate, ExecutionContext, Injectable } from "@nestjs/common";
import { Reflector } from "@nestjs/core";
import { UserRole } from "src/enums"; // Import your UserRole enum

@Injectable()
export class AuthGuard implements CanActivate {
  constructor(private reflector: Reflector) {} // Inject the Reflector to access metadata

  canActivate(context: ExecutionContext): boolean {
    const roles = this.reflector.get<UserRole[]>("roles", context.getHandler()); // Get roles from metadata
    if (!roles) {
      return true; // No roles defined, allow access
    }

    const request = context.switchToHttp().getRequest();
    const user = request.user; // Assuming your auth middleware sets the user object on the request
    if (!user) {
      return false; // No user authenticated, deny access
    }

    return roles.includes(user.role); // Check if the user has the required role
  }
}
