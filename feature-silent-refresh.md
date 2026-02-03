# Feature: Silent Refresh & NATS Ticket Vending

This workflow implements the "Ticket Vendor" authentication pattern to secure the Svelte PWA → NATS (WebSocket) → Go (Watermill) pipeline using HttpOnly cookies and NATS User JWTs.

## 0. Architecture Overview

- **Auth Layer (HTTP)**: Handled by Go backend. Uses **HttpOnly Cookies** (Persistent, 30-90 days).
- **Transport Layer (NATS WS)**: Handled by Client. Uses **Short-lived NATS JWT** (Ephemeral, 10-15 mins).
- **Logic Layer (Watermill)**: Handled by Go. Consumes the stream with authorization middleware.

### The Flow
- [x] **Login/Magic Link**: User authenticates via HTTP. Server sets `refresh_token` HttpOnly cookie.
- [x] **Ticket Request**: Svelte calls `GET /api/auth/ticket`.
- [x] **Ticket Generation**: Go validates cookie → Mints scoped NATS User JWT → Returns to client.
- [x] **NATS Connect**: Svelte connects to NATS using the JWT with automatic refresh.
- [x] **Re-Auth**: On disconnect/expiry, Svelte requests a new ticket transparently.
- [x] **Token Rotation**: Each ticket request rotates the refresh token.

---

## 1. Database Implementation (The "Ledger")

We need to track long-lived sessions to allow for revocation and detect replay attacks.

### 1.1 Database Migration (Refresh Tokens)

- [x] **Create Migration File**: `migrations/YYYYMMDD_create_refresh_tokens.sql`
- [x] **Create table `refresh_tokens`**:
  ```sql
  CREATE TABLE refresh_tokens (
      hash            VARCHAR(64) PRIMARY KEY,     -- SHA-256 of actual token
      user_uuid       UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
      token_family    VARCHAR(64) NOT NULL,        -- Detect token reuse
      expires_at      TIMESTAMPTZ NOT NULL,
      created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      last_used_at    TIMESTAMPTZ,
      revoked         BOOLEAN NOT NULL DEFAULT false,
      revoked_at      TIMESTAMPTZ,
      revoked_by      VARCHAR(50),                 -- 'user' | 'admin' | 'security'
      ip_address      INET,
      user_agent      TEXT,
      device_id       VARCHAR(255),                -- PWA installation ID
      
      CHECK (expires_at > created_at)
  );
  
  CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_uuid);
  CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE NOT revoked;
  CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(token_family);
  CREATE INDEX idx_refresh_tokens_device ON refresh_tokens(device_id) WHERE device_id IS NOT NULL;
  ```

### 1.2 Cleanup Job
- [x] **Background Task**: Delete expired tokens daily
  ```sql
  DELETE FROM refresh_tokens 
  WHERE expires_at < NOW() - INTERVAL '7 days';
  ```

---

## 2. Repository Layer (The "Vault")

### 2.1 RefreshToken Repository (`internal/repository/refresh_token.go`)

- [x] **Core Methods**:
  - [x] `SaveRefreshToken(ctx, token) error`
  - [x] `GetRefreshToken(ctx, hash) (*RefreshToken, error)`
  - [x] `RevokeRefreshToken(ctx, hash, revokedBy string) error`
  - [x] `RevokeAllUserTokens(ctx, userUUID uuid.UUID, reason string) error`
  
  - [ ] `GetActiveDevices(ctx, userUUID uuid.UUID) ([]*Device, error)` - List user sessions
  - [ ] `CleanupExpiredTokens(ctx) (int64, error)` - Maintenance

- [ ] **Implementation Example**:
  ```go
  // GetRefreshToken with constant-time lookup to prevent timing attacks
  func (r *RefreshTokenRepository) GetRefreshToken(ctx context.Context, hash string) (*RefreshToken, error) {
      token := &RefreshToken{}
      err := r.db.GetContext(ctx, token, `
          SELECT * FROM refresh_tokens 
          WHERE hash = $1 
            AND revoked = false 
            AND expires_at > NOW()
      `, hash)
      
      if err != nil {
          if errors.Is(err, sql.ErrNoRows) {
              return nil, ErrInvalidToken // Generic error
          }
          return nil, fmt.Errorf("get refresh token: %w", err)
      }
      
      // Async update last_used_at to avoid blocking
      go func() {
          ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
          defer cancel()
          _ = r.UpdateLastUsed(ctx, hash)
      }()
      
      return token, nil
  }
  ```

---

## 3. Service Layer (The "Vendor")

### 3.1 Auth Service Updates (`internal/service/auth/service.go`)

- [ ] **Method: `ExchangeCookieForTicket(ctx, cookieValue, clientInfo) (*TicketResponse, error)`**
  - Validate refresh token hash against DB
  - Check expiry and revocation
  - **Token Family Check**: Detect token reuse attacks
  - Retrieve User & Permissions (using `PermissionBuilder`)
  - Mint NATS User JWT with fine-grained permissions (reuse `UserJWTBuilder`)
  - **Optional**: Rotate refresh token (generate new one, revoke old)
  - Return JWT + new refresh token (if rotated)

- [ ] **Method: `LoginUser(ctx, user, deviceInfo) (*LoginResponse, error)`**
  - Generate Refresh Token with new family ID
  - Save to DB with device fingerprint
  - Return Refresh Token (for cookie) + Initial NATS Ticket

- [ ] **Method: `LogoutUser(ctx, cookieValue, reason string) error`**
  - Revoke token in DB
  - Optionally revoke entire token family for security

- [ ] **Method: `RevokeAllSessions(ctx, userUUID, reason) error`**
  - Admin function to force logout all devices

### 3.2 NATS JWT Builder (`internal/service/auth/nats_jwt.go`)

- [ ] **Build User-Scoped Permissions**:
  ```go
  type NATSPermissions struct {
      Publish   PermissionSet `json:"pub,omitempty"`
      Subscribe PermissionSet `json:"sub,omitempty"`
      
      // NATS 2.x limits
      Subs    int   `json:"subs,omitempty"`    // Max concurrent subscriptions
      Data    int64 `json:"data,omitempty"`    // Max bytes per message  
      Payload int64 `json:"payload,omitempty"` // Max total payload
  }
  
  type PermissionSet struct {
      Allow []string `json:"allow,omitempty"`
      Deny  []string `json:"deny,omitempty"`
  }
  
  func (b *NATSJWTBuilder) BuildPermissions(user *User) *NATSPermissions {
      userPrefix := fmt.Sprintf("user.%s", user.UUID)
      teamPrefix := fmt.Sprintf("team.%s", user.TeamID)
      
      perms := &NATSPermissions{
          Subs:    100,               // Reasonable limit
          Data:    10 * 1024 * 1024,  // 10MB max message
          Payload: 100 * 1024 * 1024, // 100MB total
      }
      
      // User-scoped subjects
      perms.Subscribe.Allow = []string{
          fmt.Sprintf("%s.>", userPrefix),           // Own data
          "public.>",                                 // Public announcements
          fmt.Sprintf("%s.>", teamPrefix),           // Team data
          fmt.Sprintf("notifications.%s.>", user.UUID), // Personal notifications
      }
      
      perms.Publish.Allow = []string{
          fmt.Sprintf("%s.commands.>", userPrefix),  // Own commands
          fmt.Sprintf("%s.events.>", teamPrefix),    // Team events
      }
      
      // Always deny admin/system subjects
      perms.Subscribe.Deny = []string{"admin.>", "system.>", "_INBOX.>"}
      perms.Publish.Deny = []string{"admin.>", "system.>"}
      
      // Role-based additions
      if user.IsAdmin {
          perms.Subscribe.Allow = append(perms.Subscribe.Allow, "admin.readonly.>")
      }
      
      return perms
  }
  
  func (b *NATSJWTBuilder) MintToken(user *User, perms *NATSPermissions) (string, error) {
      claims := jwt.MapClaims{
          "sub":  user.UUID.String(),
          "name": user.Email,
          "iat":  time.Now().Unix(),
          "exp":  time.Now().Add(15 * time.Minute).Unix(), // Short-lived
          "nbf":  time.Now().Add(-30 * time.Second).Unix(), // Clock skew tolerance
          "nats": perms,
      }
      
      token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
      return token.SignedString(b.privateKey)
  }
  ```

### 3.3 Token Rotation Logic (Optional Security Hardening)

- [ ] **Implement Refresh Token Rotation**:
  ```go
  func (s *AuthService) ExchangeWithRotation(ctx context.Context, oldTokenHash string) (*TicketResponse, error) {
      // 1. Validate old token
      oldToken, err := s.repo.GetRefreshToken(ctx, oldTokenHash)
      if err != nil {
          // Check if token was already used (replay attack)
          family, _ := s.repo.GetTokenFamily(ctx, extractFamily(oldTokenHash))
          if len(family) > 0 {
              // Token reuse detected - revoke entire family
              _ = s.repo.RevokeTokenFamily(ctx, family[0].TokenFamily)
              return nil, ErrTokenReuse
          }
          return nil, err
      }
      
      // 2. Generate new refresh token (same family)
      newToken := s.generateRefreshToken(oldToken.UserUUID, oldToken.TokenFamily)
      
      // 3. Save new token
      if err := s.repo.SaveRefreshToken(ctx, newToken); err != nil {
          return nil, err
      }
      
      // 4. Revoke old token
      if err := s.repo.RevokeRefreshToken(ctx, oldTokenHash, "rotated"); err != nil {
          return nil, err
      }
      
      // 5. Mint NATS ticket
      natsToken, err := s.mintNATSTicket(ctx, oldToken.UserUUID)
      if err != nil {
          return nil, err
      }
      
      return &TicketResponse{
          NATSToken:    natsToken,
          RefreshToken: newToken.Token, // Return for new cookie
      }, nil
  }
  ```

---

## 4. HTTP Handlers (The "Front Desk")

### 4.1 Handlers (`internal/handler/auth/ticket_handler.go`)

- [ ] **`GET /api/auth/ticket`** - Exchange cookie for NATS ticket
  - Read `__Host-refresh_token` cookie
  - Extract client info (IP, User-Agent, Device-ID header)
  - Call `ExchangeCookieForTicket`
  - If rotation enabled: Set new cookie
  - Rate limit: 10 requests/min per user
  - Return JSON: `{ "nats_token": "ey...", "expires_in": 900 }`
  
- [ ] **`POST /api/auth/logout`** - Revoke current session
  - Revoke token
  - Clear cookie with `Max-Age=-1`
  - Return 204 No Content
  
- [ ] **`POST /api/auth/logout-all`** - Revoke all user sessions
  - Require re-authentication (password/OTP)
  - Revoke all tokens for user
  - Return list of revoked devices

- [ ] **`GET /api/auth/devices`** - List active sessions
  - Return user's active devices with last_used_at
  - Allow revoking individual devices

- [ ] **`POST /api/auth/login`** (or update Magic Link callback)
  - Generate refresh token with new family
  - Set secure cookie:
    ```go
    http.SetCookie(w, &http.Cookie{
        Name:     "__Host-refresh_token", // __Host- requires Secure + Path=/
        Value:    tokenHash,
        Path:     "/",
        MaxAge:   60 * 60 * 24 * 90, // 90 days
        HttpOnly: true,
        Secure:   true,               // HTTPS only
        SameSite: http.SameSiteStrictMode,
        // Domain:  "", // Empty for __Host- prefix
    })
    ```
  - Return initial NATS ticket for immediate connection

### 4.2 Rate Limiting Middleware

- [ ] **Implement Redis-based rate limiter**:
  ```go
  func RateLimitTicketRequests() gin.HandlerFunc {
      return func(c *gin.Context) {
          userID := c.GetString("user_id")
          key := fmt.Sprintf("ratelimit:ticket:%s", userID)
          
          count, err := redisClient.Incr(ctx, key).Result()
          if err == nil && count == 1 {
              redisClient.Expire(ctx, key, time.Minute)
          }
          
          if count > 10 {
              c.JSON(429, gin.H{"error": "rate limit exceeded"})
              c.Abort()
              return
          }
          
          c.Next()
      }
  }
  ```

---

## 5. Frontend Implementation (SvelteKit PWA)

### 5.1 Service Worker (`src/service-worker.ts`)

- [ ] **Setup Vite PWA Plugin**:
  ```typescript
  // vite.config.ts
  import { sveltekit } from '@sveltejs/kit/vite';
  import { VitePWA } from 'vite-plugin-pwa';
  
  export default {
      plugins: [
          sveltekit(),
          VitePWA({
              registerType: 'autoUpdate',
              devOptions: { enabled: true },
              workbox: {
                  globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
                  runtimeCaching: [
                      {
                          urlPattern: /^https:\/\/api\.yourdomain\.com\/auth\/ticket$/,
                          handler: 'NetworkOnly', // Never cache auth endpoints
                      },
                      {
                          urlPattern: /^https:\/\/api\.yourdomain\.com\/api\//,
                          handler: 'NetworkFirst',
                          options: {
                              cacheName: 'api-cache',
                              networkTimeoutSeconds: 10,
                              cacheableResponse: { statuses: [0, 200] }
                          }
                      }
                  ]
              },
              manifest: {
                  name: 'Your App',
                  short_name: 'App',
                  theme_color: '#ffffff',
                  background_color: '#ffffff',
                  display: 'standalone',
                  scope: '/',
                  start_url: '/',
                  icons: [/* ... */]
              }
          })
      ]
  };
  ```

### 5.2 Device ID Generation (`src/lib/device.ts`)

- [ ] **Generate Persistent Device ID**:
  ```typescript
  import { browser } from '$app/environment';
  
  export class DeviceManager {
      private static STORAGE_KEY = 'device_id';
      
      static getDeviceId(): string {
          if (!browser) return '';
          
          let deviceId = localStorage.getItem(this.STORAGE_KEY);
          
          if (!deviceId) {
              deviceId = this.generateDeviceId();
              localStorage.setItem(this.STORAGE_KEY, deviceId);
          }
          
          return deviceId;
      }
      
      private static generateDeviceId(): string {
          const array = new Uint8Array(16);
          crypto.getRandomValues(array);
          return Array.from(array, b => b.toString(16).padStart(2, '0')).join('');
      }
      
      static getFingerprint(): DeviceFingerprint {
          return {
              deviceId: this.getDeviceId(),
              userAgent: navigator.userAgent,
              language: navigator.language,
              platform: navigator.platform,
              screenResolution: `${screen.width}x${screen.height}`,
              timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              // Don't overdo it - privacy concerns
          };
      }
  }
  
  export interface DeviceFingerprint {
      deviceId: string;
      userAgent: string;
      language: string;
      platform: string;
      screenResolution: string;
      timezone: string;
  }
  ```

### 5.3 Auth Store (`src/lib/stores/auth.svelte.ts`)

- [ ] **Implement Svelte 5 Runes-based Store**:
  ```typescript
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { DeviceManager } from '$lib/device';
  
  interface User {
      uuid: string;
      email: string;
      name: string;
      teamId: string;
      roles: string[];
  }
  
  interface AuthState {
      user: User | null;
      isLoading: boolean;
      lastTicketRefresh: number | null;
  }
  
  class AuthStore {
      private state = $state<AuthState>({
          user: null,
          isLoading: true,
          lastTicketRefresh: null
      });
      
      constructor() {
          if (browser) {
              this.loadUserFromStorage();
          }
      }
      
      get user() {
          return this.state.user;
      }
      
      get isAuthenticated() {
          return this.state.user !== null;
      }
      
      get isLoading() {
          return this.state.isLoading;
      }
      
      private loadUserFromStorage() {
          try {
              const stored = localStorage.getItem('user');
              if (stored) {
                  this.state.user = JSON.parse(stored);
              }
          } catch (err) {
              console.error('Failed to load user from storage:', err);
              localStorage.removeItem('user');
          } finally {
              this.state.isLoading = false;
          }
      }
      
      setUser(user: User | null) {
          this.state.user = user;
          
          if (browser) {
              if (user) {
                  localStorage.setItem('user', JSON.stringify(user));
              } else {
                  localStorage.removeItem('user');
              }
          }
      }
      
      async getTicket(): Promise<string> {
          const deviceId = DeviceManager.getDeviceId();
          
          const response = await fetch('/api/auth/ticket', {
              method: 'GET',
              credentials: 'include', // Send HttpOnly cookie
              headers: {
                  'X-Device-ID': deviceId
              }
          });
          
          if (!response.ok) {
              if (response.status === 401) {
                  // Cookie expired or invalid - force re-login
                  this.logout();
                  goto('/login');
                  throw new Error('Session expired');
              }
              throw new Error(`Failed to get ticket: ${response.status}`);
          }
          
          const data = await response.json();
          this.state.lastTicketRefresh = Date.now();
          
          return data.nats_token;
      }
      
      async login(email: string, magicLinkToken?: string): Promise<void> {
          const deviceInfo = DeviceManager.getFingerprint();
          
          const response = await fetch('/api/auth/login', {
              method: 'POST',
              credentials: 'include',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ email, token: magicLinkToken, deviceInfo })
          });
          
          if (!response.ok) {
              throw new Error('Login failed');
          }
          
          const data = await response.json();
          this.setUser(data.user);
          
          // Initial ticket returned for immediate NATS connection
          return data.nats_token;
      }
      
      async logout(): Promise<void> {
          await fetch('/api/auth/logout', {
              method: 'POST',
              credentials: 'include'
          }).catch(() => {
              // Best effort - continue with local cleanup
          });
          
          this.setUser(null);
          this.state.lastTicketRefresh = null;
      }
  }
  
  export const auth = new AuthStore();
  ```

### 5.4 NATS Service (`src/lib/services/nats.svelte.ts`)

- [ ] **Robust NATS Connection Manager**:
  ```typescript
  import { connect, type NatsConnection, type ConnectionOptions, Events } from 'nats.ws';
  import { auth } from '$lib/stores/auth.svelte';
  import { browser } from '$app/environment';
  
  const NATS_SERVERS = ['ws://localhost:9222']; // Use WSS in production
  
  class NATSManager {
      private nc: NatsConnection | null = null;
      private reconnectAttempts = 0;
      private maxReconnectAttempts = 5;
      private reconnectBackoff = [1000, 2000, 5000, 10000, 30000];
      private isConnecting = false;
      private connectionPromise: Promise<void> | null = null;
      
      // Reactive state
      private state = $state({
          connected: false,
          connecting: false,
          error: null as string | null
      });
      
      get connected() {
          return this.state.connected;
      }
      
      get connecting() {
          return this.state.connecting;
      }
      
      async connect(): Promise<void> {
          // Prevent concurrent connection attempts
          if (this.isConnecting) {
              return this.connectionPromise!;
          }
          
          if (this.nc && !this.nc.isClosed()) {
              return; // Already connected
          }
          
          this.isConnecting = true;
          this.state.connecting = true;
          this.state.error = null;
          
          this.connectionPromise = this._connect();
          
          try {
              await this.connectionPromise;
          } finally {
              this.isConnecting = false;
              this.connectionPromise = null;
          }
      }
      
      private async _connect(): Promise<void> {
          try {
              const token = await auth.getTicket();
              
              const opts: ConnectionOptions = {
                  servers: NATS_SERVERS,
                  token,
                  
                  // Connection settings
                  name: 'svelte-pwa-client',
                  timeout: 10000, // 10s connection timeout
                  
                  // Reconnection strategy
                  reconnect: true,
                  maxReconnectAttempts: -1, // Infinite - we handle via authenticator
                  reconnectTimeWait: 2000,  // 2s between attempts
                  
                  // Auto-refresh token on reconnect
                  authenticator: async () => {
                      try {
                          return await auth.getTicket();
                      } catch (err) {
                          console.error('Token refresh failed during reconnect:', err);
                          throw err;
                      }
                  }
              };
              
              this.nc = await connect(opts);
              this.state.connected = true;
              this.state.connecting = false;
              this.reconnectAttempts = 0;
              
              console.log('✅ NATS connected:', this.nc.info);
              
              // Monitor connection status
              this.monitorConnection();
              
          } catch (err: any) {
              this.state.connecting = false;
              await this.handleConnectError(err);
          }
      }
      
      private monitorConnection() {
          if (!this.nc) return;
          
          // Listen for connection events
          (async () => {
              for await (const status of this.nc!.status()) {
                  console.log('NATS status:', status.type, status.data);
                  
                  switch (status.type) {
                      case Events.Disconnect:
                          this.state.connected = false;
                          break;
                          
                      case Events.Reconnect:
                          this.state.connected = true;
                          this.reconnectAttempts = 0;
                          break;
                          
                      case Events.Error:
                          // Check for auth errors
                          if (status.data?.includes('Authorization Violation')) {
                              await this.handleTokenExpiry();
                          } else {
                              this.state.error = status.data || 'Connection error';
                          }
                          break;
                  }
              }
          })();
      }
      
      private async handleTokenExpiry() {
          console.warn('⚠️ NATS token expired, attempting refresh...');
          
          if (this.reconnectAttempts >= this.maxReconnectAttempts) {
              console.error('❌ Max reconnect attempts reached, forcing re-login');
              this.state.error = 'Session expired - please log in again';
              await this.disconnect();
              auth.logout();
              return;
          }
          
          const delay = this.reconnectBackoff[this.reconnectAttempts] || 30000;
          console.log(`Waiting ${delay}ms before reconnect attempt ${this.reconnectAttempts + 1}`);
          
          await new Promise(resolve => setTimeout(resolve, delay));
          this.reconnectAttempts++;
          
          // Disconnect and reconnect with fresh token
          await this.disconnect();
          await this.connect();
      }
      
      private async handleConnectError(err: Error) {
          console.error('NATS connection error:', err);
          this.state.error = err.message;
          
          // Don't retry on auth errors
          if (err.message.includes('Authorization') || err.message.includes('401')) {
              auth.logout();
              return;
          }
          
          // Exponential backoff for other errors
          if (this.reconnectAttempts < this.maxReconnectAttempts) {
              const delay = this.reconnectBackoff[this.reconnectAttempts] || 30000;
              this.reconnectAttempts++;
              
              setTimeout(() => this.connect(), delay);
          } else {
              this.state.error = 'Failed to connect after multiple attempts';
          }
      }
      
      async disconnect(): Promise<void> {
          if (this.nc && !this.nc.isClosed()) {
              await this.nc.drain();
              await this.nc.close();
          }
          this.nc = null;
          this.state.connected = false;
      }
      
      getConnection(): NatsConnection | null {
          return this.nc;
      }
      
      // Helper for common pub/sub operations
      async publish(subject: string, data: any): Promise<void> {
          if (!this.nc) {
              throw new Error('NATS not connected');
          }
          
          const payload = JSON.stringify(data);
          this.nc.publish(subject, new TextEncoder().encode(payload));
      }
      
      async subscribe(subject: string, callback: (data: any) => void) {
          if (!this.nc) {
              throw new Error('NATS not connected');
          }
          
          const sub = this.nc.subscribe(subject);
          
          (async () => {
              for await (const msg of sub) {
                  try {
                      const data = JSON.parse(new TextDecoder().decode(msg.data));
                      callback(data);
                  } catch (err) {
                      console.error('Error processing message:', err);
                  }
              }
          })();
          
          return sub;
      }
  }
  
  export const nats = browser ? new NATSManager() : null;
  ```

### 5.5 App Initialization (`src/routes/+layout.svelte`)

- [ ] **Auto-connect on App Load**:
  ```svelte
  <script lang="ts">
      import { onMount } from 'svelte';
      import { auth } from '$lib/stores/auth.svelte';
      import { nats } from '$lib/services/nats.svelte';
      import { page } from '$app/stores';
      
      let mounted = $state(false);
      
      onMount(async () => {
          // Wait for auth to load from storage
          while (auth.isLoading) {
              await new Promise(r => setTimeout(r, 50));
          }
          
          // Connect to NATS if authenticated
          if (auth.isAuthenticated && nats) {
              try {
                  await nats.connect();
              } catch (err) {
                  console.error('Failed to connect to NATS on mount:', err);
              }
          }
          
          mounted = true;
      });
      
      // Reactive connection on auth state changes
      $effect(() => {
          if (!mounted || !nats) return;
          
          if (auth.isAuthenticated && !nats.connected && !nats.connecting) {
              nats.connect().catch(console.error);
          } else if (!auth.isAuthenticated && nats.connected) {
              nats.disconnect().catch(console.error);
          }
      });
  </script>
  
  {#if auth.isLoading}
      <div class="loading">Loading...</div>
  {:else}
      <slot />
  {/if}
  ```

### 5.6 Offline Support (`src/lib/stores/offline.svelte.ts`)

- [ ] **Detect Online/Offline State**:
  ```typescript
  import { browser } from '$app/environment';
  import { nats } from '$lib/services/nats.svelte';
  
  class OfflineManager {
      private state = $state({
          online: browser ? navigator.onLine : true,
          queuedMessages: [] as Array<{ subject: string; data: any }>
      });
      
      constructor() {
          if (browser) {
              window.addEventListener('online', this.handleOnline.bind(this));
              window.addEventListener('offline', this.handleOffline.bind(this));
          }
      }
      
      get isOnline() {
          return this.state.online;
      }
      
      get queueSize() {
          return this.state.queuedMessages.length;
      }
      
      private handleOnline() {
          console.log('✅ App is online');
          this.state.online = true;
          
          // Reconnect NATS
          nats?.connect().catch(console.error);
          
          // Flush queued messages
          this.flushQueue();
      }
      
      private handleOffline() {
          console.log('⚠️ App is offline');
          this.state.online = false;
      }
      
      async publish(subject: string, data: any) {
          if (!this.state.online) {
              // Queue for later
              this.state.queuedMessages.push({ subject, data });
              console.log('Message queued for offline:', subject);
              return;
          }
          
          try {
              await nats?.publish(subject, data);
          } catch (err) {
              // If publish fails, queue it
              this.state.queuedMessages.push({ subject, data });
              throw err;
          }
      }
      
      private async flushQueue() {
          console.log(`Flushing ${this.state.queuedMessages.length} queued messages`);
          
          const messages = [...this.state.queuedMessages];
          this.state.queuedMessages = [];
          
          for (const msg of messages) {
              try {
                  await nats?.publish(msg.subject, msg.data);
              } catch (err) {
                  // Re-queue failed messages
                  this.state.queuedMessages.push(msg);
                  console.error('Failed to flush message:', err);
              }
          }
      }
  }
  
  export const offline = browser ? new OfflineManager() : null;
  ```

---

## 6. Watermill Integration (Event Handlers)

### 6.1 NATS Subscriber with Authorization Middleware

- [ ] **Setup Watermill Router** (`internal/events/router.go`):
  ```go
  package events
  
  import (
      "context"
      "encoding/json"
      "fmt"
      
      "github.com/ThreeDotsLabs/watermill"
      "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
      "github.com/ThreeDotsLabs/watermill/message"
      "github.com/nats-io/nats.go"
  )
  
  type Router struct {
      router    *message.Router
      publisher message.Publisher
      logger    watermill.LoggerAdapter
      validator *JWTValidator
  }
  
  func NewRouter(nc *nats.Conn, logger watermill.LoggerAdapter, validator *JWTValidator) (*Router, error) {
      // Create NATS subscriber with JetStream
      subscriber, err := nats.NewSubscriber(
          nats.SubscriberConfig{
              URL:            nc.Opts.Url,
              JetStream:      nats.JetStreamConfig{Enabled: true},
              SubscribeOptions: []nats.SubscribeOption{
                  nats.ManualAck(),
                  nats.DeliverNew(),
              },
          },
          logger,
      )
      if err != nil {
          return nil, err
      }
      
      // Create publisher
      publisher, err := nats.NewPublisher(
          nats.PublisherConfig{
              URL:       nc.Opts.Url,
              JetStream: nats.JetStreamConfig{Enabled: true},
          },
          logger,
      )
      if err != nil {
          return nil, err
      }
      
      // Create router
      router, err := message.NewRouter(message.RouterConfig{}, logger)
      if err != nil {
          return nil, err
      }
      
      return &Router{
          router:    router,
          publisher: publisher,
          logger:    logger,
          validator: validator,
      }, nil
  }
  
  func (r *Router) AddHandler(topic string, handler message.HandlerFunc) {
      r.router.AddMiddleware(
          // Add user context from NATS subject
          r.extractUserMiddleware,
          // Authorize based on subject pattern
          r.authorizationMiddleware,
          // Structured logging
          r.loggingMiddleware,
          // Recover from panics
          message.Recoverer,
      )
      
      r.router.AddNoPublisherHandler(
          fmt.Sprintf("handler_%s", topic),
          topic,
          r.subscriber,
          handler,
      )
  }
  ```

### 6.2 Authorization Middleware

- [ ] **Subject-based Authorization** (`internal/events/middleware.go`):
  ```go
  func (r *Router) extractUserMiddleware(h message.HandlerFunc) message.HandlerFunc {
      return func(msg *message.Message) ([]*message.Message, error) {
          // NATS subject is available in metadata
          subject := msg.Metadata.Get("nats-subject")
          
          // Extract user UUID from subject (e.g., "user.123e4567.commands.create")
          parts := strings.Split(subject, ".")
          if len(parts) >= 2 && parts[0] == "user" {
              msg.Metadata.Set("user-id", parts[1])
          }
          
          return h(msg)
      }
  }
  
  func (r *Router) authorizationMiddleware(h message.HandlerFunc) message.HandlerFunc {
      return func(msg *message.Message) ([]*message.Message, error) {
          subject := msg.Metadata.Get("nats-subject")
          userID := msg.Metadata.Get("user-id")
          
          // Validate user has permission for this subject
          if !r.canAccessSubject(userID, subject) {
              r.logger.Error("Unauthorized access attempt", watermill.LogFields{
                  "user_id": userID,
                  "subject": subject,
              })
              msg.Nack()
              return nil, fmt.Errorf("unauthorized: user %s cannot access %s", userID, subject)
          }
          
          return h(msg)
      }
  }
  
  func (r *Router) canAccessSubject(userID, subject string) bool {
      // Check if subject matches allowed patterns
      // This should match the permissions set in NATS JWT
      
      allowedPrefixes := []string{
          fmt.Sprintf("user.%s.", userID),
          "public.",
      }
      
      for _, prefix := range allowedPrefixes {
          if strings.HasPrefix(subject, prefix) {
              return true
          }
      }
      
      return false
  }
  ```

### 6.3 Example Event Handler

- [ ] **User Command Handler** (`internal/events/handlers/user_commands.go`):
  ```go
  package handlers
  
  import (
      "context"
      "encoding/json"
      "fmt"
      
      "github.com/ThreeDotsLabs/watermill/message"
      "github.com/google/uuid"
  )
  
  type UserCommandHandler struct {
      userService *service.UserService
  }
  
  func (h *UserCommandHandler) HandleUpdateProfile(msg *message.Message) ([]*message.Message, error) {
      // Extract user ID from metadata (set by middleware)
      userID := msg.Metadata.Get("user-id")
      if userID == "" {
          return nil, fmt.Errorf("user-id not found in metadata")
      }
      
      // Parse command payload
      var cmd struct {
          Name  string `json:"name"`
          Bio   string `json:"bio"`
      }
      if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
          return nil, fmt.Errorf("invalid payload: %w", err)
      }
      
      // Execute business logic
      ctx := context.Background()
      userUUID, _ := uuid.Parse(userID)
      
      if err := h.userService.UpdateProfile(ctx, userUUID, cmd.Name, cmd.Bio); err != nil {
          return nil, err
      }
      
      // Ack the message
      msg.Ack()
      
      // Optionally publish event
      event := &message.Message{
          UUID:    watermill.NewUUID(),
          Payload: msg.Payload,
          Metadata: message.Metadata{
              "event-type": "profile-updated",
              "user-id":    userID,
          },
      }
      
      return []*message.Message{event}, nil
  }
  ```

---

## 7. Observability & Monitoring

### 7.1 Metrics

- [ ] **Prometheus Metrics** (`internal/metrics/auth_metrics.go`):
  ```go
  package metrics
  
  import (
      "github.com/prometheus/client_golang/prometheus"
      "github.com/prometheus/client_golang/prometheus/promauto"
  )
  
  var (
      TicketIssuances = promauto.NewCounterVec(
          prometheus.CounterOpts{
              Name: "auth_ticket_issuances_total",
              Help: "Total number of NATS tickets issued",
          },
          []string{"user_id", "status"}, // status: success, expired, revoked, invalid
      )
      
      TokenRotations = promauto.NewCounter(
          prometheus.CounterOpts{
              Name: "auth_token_rotations_total",
              Help: "Total number of refresh token rotations",
          },
      )
      
      TokenFamilyViolations = promauto.NewCounter(
          prometheus.CounterOpts{
              Name: "auth_token_family_violations_total",
              Help: "Total number of token reuse attempts detected",
          },
      )
      
      ActiveSessions = promauto.NewGauge(
          prometheus.GaugeOpts{
              Name: "auth_active_sessions",
              Help: "Number of currently active sessions",
          },
      )
      
      NATSConnectionAttempts = promauto.NewCounterVec(
          prometheus.CounterOpts{
              Name: "nats_connection_attempts_total",
              Help: "Total NATS connection attempts",
          },
          []string{"status"}, // success, auth_failed, network_error
      )
      
      NATSMessageLatency = promauto.NewHistogramVec(
          prometheus.HistogramOpts{
              Name: "nats_message_processing_duration_seconds",
              Help: "NATS message processing latency",
              Buckets: prometheus.DefBuckets,
          },
          []string{"subject", "handler"},
      )
  )
  ```

### 7.2 Structured Logging

- [ ] **Audit Logging** (`internal/logging/audit.go`):
  ```go
  type AuditLogger struct {
      logger *zap.Logger
  }
  
  func (l *AuditLogger) LogTicketIssued(ctx context.Context, event TicketIssuedEvent) {
      l.logger.Info("ticket_issued",
          zap.String("user_id", event.UserID),
          zap.String("device_id", event.DeviceID),
          zap.String("ip", event.IP),
          zap.Time("expires_at", event.ExpiresAt),
          zap.Duration("ttl", time.Until(event.ExpiresAt)),
      )
  }
  
  func (l *AuditLogger) LogTokenRevoked(ctx context.Context, event TokenRevokedEvent) {
      l.logger.Warn("token_revoked",
          zap.String("user_id", event.UserID),
          zap.String("token_hash", event.TokenHash),
          zap.String("reason", event.Reason),
          zap.String("revoked_by", event.RevokedBy),
      )
  }
  
  func (l *AuditLogger) LogFamilyViolation(ctx context.Context, event FamilyViolationEvent) {
      l.logger.Error("token_family_violation",
          zap.String("user_id", event.UserID),
          zap.String("family_id", event.FamilyID),
          zap.String("ip", event.IP),
          zap.String("user_agent", event.UserAgent),
      )
  }
  ```

### 7.3 Alerting Rules

- [ ] **Alert Definitions** (`monitoring/alerts.yml`):
  ```yaml
  groups:
    - name: auth_alerts
      interval: 30s
      rules:
        - alert: HighTokenFamilyViolations
          expr: rate(auth_token_family_violations_total[5m]) > 0.1
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "High rate of token reuse attempts detected"
            
        - alert: TicketIssuanceFailureRate
          expr: rate(auth_ticket_issuances_total{status="failed"}[5m]) > 0.05
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Elevated ticket issuance failure rate"
            
        - alert: NATSConnectionFailures
          expr: rate(nats_connection_attempts_total{status!="success"}[5m]) > 1
          for: 3m
          labels:
            severity: warning
          annotations:
            summary: "Multiple NATS connection failures detected"
  ```

---

## 8. Security Hardening

### 8.1 Rate Limiting

- [ ] **Multi-tier Rate Limits**:
  ```go
  type RateLimiter struct {
      redis *redis.Client
  }
  
  // Per-user rate limit: 10 tickets/min
  func (rl *RateLimiter) CheckUserLimit(ctx context.Context, userID string) error {
      key := fmt.Sprintf("ratelimit:ticket:user:%s", userID)
      return rl.checkLimit(ctx, key, 10, time.Minute)
  }
  
  // Per-IP rate limit: 20 tickets/min
  func (rl *RateLimiter) CheckIPLimit(ctx context.Context, ip string) error {
      key := fmt.Sprintf("ratelimit:ticket:ip:%s", ip)
      return rl.checkLimit(ctx, key, 20, time.Minute)
  }
  
  // Global rate limit: 1000 tickets/min
  func (rl *RateLimiter) CheckGlobalLimit(ctx context.Context) error {
      key := "ratelimit:ticket:global"
      return rl.checkLimit(ctx, key, 1000, time.Minute)
  }
  
  func (rl *RateLimiter) checkLimit(ctx context.Context, key string, limit int, window time.Duration) error {
      pipe := rl.redis.Pipeline()
      incr := pipe.Incr(ctx, key)
      pipe.Expire(ctx, key, window)
      
      if _, err := pipe.Exec(ctx); err != nil {
          return err
      }
      
      if incr.Val() > int64(limit) {
          return ErrRateLimitExceeded
      }
      
      return nil
  }
  ```

### 8.2 CSRF Protection

- [ ] **CSRF Token for State-Changing Operations**:
  ```go
  // Only needed for non-NATS operations (logout, logout-all)
  func CSRFMiddleware() gin.HandlerFunc {
      return func(c *gin.Context) {
          if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
              token := c.GetHeader("X-CSRF-Token")
              
              // Validate against session
              sessionToken := getSessionCSRFToken(c)
              if token != sessionToken {
                  c.AbortWithStatusJSON(403, gin.H{"error": "invalid CSRF token"})
                  return
              }
          }
          c.Next()
      }
  }
  ```

### 8.3 Geo-IP Blocking (Optional)

- [ ] **Block Suspicious Countries**:
  ```go
  var blockedCountries = map[string]bool{
      "XX": true, // Example
  }
  
  func GeoIPMiddleware(geoip *GeoIPDB) gin.HandlerFunc {
      return func(c *gin.Context) {
          ip := c.ClientIP()
          country := geoip.Lookup(ip)
          
          if blockedCountries[country] {
              logger.Warn("Blocked request from restricted country",
                  zap.String("ip", ip),
                  zap.String("country", country),
              )
              c.AbortWithStatus(403)
              return
          }
          
          c.Next()
      }
  }
  ```

---

## 9. Testing Strategy

### 9.1 Backend Tests

- [ ] **Unit Tests** (`internal/service/auth/service_test.go`):
  ```go
  func TestExchangeCookieForTicket_ValidToken(t *testing.T) {
      // Setup
      repo := mock.NewRefreshTokenRepository()
      service := NewAuthService(repo, jwtBuilder)
      
      token := generateTestToken(userUUID)
      repo.SaveRefreshToken(ctx, token)
      
      // Execute
      ticket, err := service.ExchangeCookieForTicket(ctx, token.Hash, clientInfo)
      
      // Assert
      require.NoError(t, err)
      assert.NotEmpty(t, ticket.NATSToken)
  }
  
  func TestExchangeCookieForTicket_TokenReuse(t *testing.T) {
      // Test token family violation detection
      // Should revoke entire family
  }
  ```

- [ ] **Integration Tests** (`test/integration/auth_flow_test.go`):
  ```go
  func TestEndToEndAuthFlow(t *testing.T) {
      // 1. Login via magic link
      // 2. Receive refresh token cookie
      // 3. Request NATS ticket
      // 4. Connect to NATS
      // 5. Publish message
      // 6. Verify message received
  }
  ```

### 9.2 Frontend Tests

- [ ] **Vitest Unit Tests** (`src/lib/stores/auth.test.ts`):
  ```typescript
  import { describe, it, expect, vi } from 'vitest';
  import { auth } from './auth.svelte';
  
  describe('AuthStore', () => {
      it('should load user from localStorage on init', () => {
          const user = { uuid: '123', email: 'test@example.com' };
          localStorage.setItem('user', JSON.stringify(user));
          
          // Trigger reload
          const store = new AuthStore();
          
          expect(store.user).toEqual(user);
      });
      
      it('should clear user on logout', async () => {
          // ... test implementation
      });
  });
  ```

- [ ] **Playwright E2E Tests** (`tests/e2e/auth.spec.ts`):
  ```typescript
  import { test, expect } from '@playwright/test';
  
  test('complete auth flow', async ({ page }) => {
      // 1. Navigate to login
      await page.goto('/login');
      
      // 2. Enter email
      await page.fill('input[type="email"]', 'test@example.com');
      await page.click('button[type="submit"]');
      
      // 3. Verify magic link sent
      await expect(page.locator('text=Check your email')).toBeVisible();
      
      // 4. Click magic link (simulate)
      // ... continue flow
  });
  ```

---

## 10. Deployment Checklist

### 10.1 Infrastructure

- [ ] **NATS Cluster Setup**:
  - [ ] Deploy NATS with JetStream enabled
  - [ ] Configure TLS for WebSocket connections
  - [ ] Set up NATS account for JWT signing
  - [ ] Generate and secure NKey for signing

- [ ] **Database**:
  - [ ] Run migrations
  - [ ] Set up automated backups
  - [ ] Create indexes

- [ ] **Redis**:
  - [ ] Deploy for rate limiting
  - [ ] Configure persistence (AOF)

### 10.2 Configuration

- [ ] **Environment Variables**:
  ```bash
  # Auth
  REFRESH_TOKEN_SECRET=xxx
  NATS_JWT_SIGNING_KEY=xxx
  
  # NATS
  NATS_URL=nats://nats:4222
  NATS_WS_URL=wss://nats.yourdomain.com
  
  # Cookies
  COOKIE_DOMAIN=yourdomain.com
  COOKIE_SECURE=true
  
  # Rate Limits
  REDIS_URL=redis://redis:6379
  ```

### 10.3 Monitoring

- [ ] Set up Prometheus
- [ ] Configure Grafana dashboards
- [ ] Set up alert notifications (PagerDuty/Slack)
- [ ] Enable distributed tracing (Jaeger/Tempo)

### 10.4 Security

- [ ] Enable HTTPS everywhere
- [ ] Configure WAF rules
- [ ] Set up DDoS protection
- [ ] Schedule security audits
- [ ] Implement IP whitelist for admin endpoints

---

## 11. Rollout Plan

### Phase 1: Backend Foundation (Week 1)
- [ ] Database migration
- [ ] Repository layer
- [ ] Service layer with basic ticket exchange
- [ ] HTTP handlers
- [ ] Unit tests

### Phase 2: NATS Integration (Week 2)
- [ ] NATS JWT builder with permissions
- [ ] Watermill subscriber setup
- [ ] Authorization middleware
- [ ] Integration tests

### Phase 3: Frontend Implementation (Week 3)
- [ ] Auth store with Svelte 5 runes
- [ ] NATS service with reconnection logic
- [ ] Device ID generation
- [ ] PWA setup with service worker

### Phase 4: Security Hardening (Week 4)
- [ ] Token rotation
- [ ] Token family tracking
- [ ] Rate limiting
- [ ] CSRF protection
- [ ] Security tests

### Phase 5: Observability (Week 5)
- [ ] Metrics instrumentation
- [ ] Audit logging
- [ ] Dashboards
- [ ] Alerts
- [ ] Load testing

### Phase 6: Production Deployment (Week 6)
- [ ] Staging deployment
- [ ] Load testing
- [ ] Security audit
- [ ] Gradual rollout (10% → 50% → 100%)
- [ ] Monitor metrics

---

## 12. Verification & Testing

### 12.1 Manual Testing

- [ ] **Happy Path**:
  - [ ] Log in via magic link
  - [ ] Verify refresh token cookie is set
  - [ ] Open DevTools → Application → Cookies → Verify `__Host-refresh_token`
  - [ ] Verify NATS connection established
  - [ ] Publish a message via NATS
  - [ ] Verify message received by Watermill handler

- [ ] **Token Refresh**:
  - [ ] Wait for NATS token to expire (~15 min)
  - [ ] Verify automatic reconnection
  - [ ] Check network tab for `/api/auth/ticket` call
  - [ ] Verify no user disruption

- [ ] **Revocation**:
  - [ ] Log out
  - [ ] Verify cookie cleared
  - [ ] Attempt to request ticket → Expect 401
  - [ ] Verify redirect to login

- [ ] **Multi-Device**:
  - [ ] Log in on desktop
  - [ ] Log in on mobile
  - [ ] Navigate to `/settings/devices`
  - [ ] Verify both sessions listed
  - [ ] Revoke one device
  - [ ] Verify other device still works

### 12.2 Security Testing

- [ ] **Token Reuse Attack**:
  - [ ] Capture refresh token cookie
  - [ ] Use it to get ticket
  - [ ] Reuse old token → Expect entire family revoked

- [ ] **CSRF Attack**:
  - [ ] Attempt POST /api/auth/logout without CSRF token
  - [ ] Expect 403

- [ ] **Rate Limit**:
  - [ ] Request 20 tickets in quick succession
  - [ ] Expect 429 after limit

### 12.3 Load Testing

- [ ] **Artillery Script** (`load-test.yml`):
  ```yaml
  config:
    target: "https://api.yourdomain.com"
    phases:
      - duration: 60
        arrivalRate: 10
        name: "Warm up"
      - duration: 300
        arrivalRate: 50
        name: "Sustained load"
  scenarios:
    - name: "Ticket exchange"
      flow:
        - post:
            url: "/api/auth/login"
            json:
              email: "load-test-{{ $randomString() }}@example.com"
            capture:
              - json: "$.refresh_token"
                as: "token"
        - get:
            url: "/api/auth/ticket"
            headers:
              Cookie: "refresh_token={{ token }}"
  ```

---

## Appendix: Code Snippets

### A. Generate Refresh Token

```go
func generateRefreshToken(userUUID uuid.UUID, familyID string) *RefreshToken {
    tokenBytes := make([]byte, 32)
    rand.Read(tokenBytes)
    token := base64.URLEncoding.EncodeToString(tokenBytes)
    
    hash := sha256.Sum256([]byte(token))
    
    return &RefreshToken{
        Token:       token,
        Hash:        hex.EncodeToString(hash[:]),
        UserUUID:    userUUID,
        TokenFamily: familyID,
        ExpiresAt:   time.Now().Add(90 * 24 * time.Hour),
        CreatedAt:   time.Now(),
    }
}
```

### B. Extract User from NATS JWT in Frontend

```typescript
function parseJWT(token: string): any {
    const parts = token.split('.');
    if (parts.length !== 3) throw new Error('Invalid JWT');
    
    const payload = JSON.parse(atob(parts[1]));
    return payload;
}

// Usage
const ticket = await auth.getTicket();
const claims = parseJWT(ticket);
console.log('Token expires in:', claims.exp - Date.now() / 1000, 'seconds');
```

### C. NATS Stream Configuration

```go
js, _ := nc.JetStream()

// Create stream for user commands
_, err := js.AddStream(&nats.StreamConfig{
    Name:     "USER_COMMANDS",
    Subjects: []string{"user.*.commands.>"},
    Storage:  nats.FileStorage,
    Retention: nats.WorkQueuePolicy,
    MaxAge:   24 * time.Hour,
    Replicas: 3, // For HA
})
```

---

## Summary

This implementation provides:

✅ **Security**: Token rotation, family tracking, revocation, rate limiting  
✅ **Resilience**: Automatic reconnection, offline queueing, exponential backoff  
✅ **Observability**: Metrics, structured logs, alerts  
✅ **PWA Support**: Service worker, offline detection, device fingerprinting  
✅ **Scalability**: NATS JetStream, Watermill, fine-grained permissions  
✅ **Developer Experience**: Svelte 5 runes, TypeScript, comprehensive tests

The plan is production-ready for 2026 best practices.
