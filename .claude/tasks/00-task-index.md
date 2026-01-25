# PWA Backend Integration Tasks

## Execution Order

```
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 1: FOUNDATION                       │
├─────────────────────────────────────────────────────────────┤
│  Task 01: JWT Package          Task 03: Event Subject Audit │
│  Agent: Sonnet                 Agent: Sonnet                │
│  ─────────────────────         ─────────────────────        │
│  Can run in PARALLEL                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 2: CONFIGURATION                    │
├─────────────────────────────────────────────────────────────┤
│  Task 05: Config & Environment                              │
│  Agent: Haiku                                               │
│  Depends on: Task 01                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 3: CORE SERVICES                    │
├─────────────────────────────────────────────────────────────┤
│  Task 02: Auth Callout Module  Task 04: Request/Reply       │
│  Agent: Opus                   Agent: Sonnet                │
│  Depends: 01, 05               Depends: 03                  │
│  ─────────────────────         ─────────────────────        │
│  Can run in PARALLEL                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    PHASE 4: INTEGRATION                      │
├─────────────────────────────────────────────────────────────┤
│  Task 06: Magic Link Command Support                        │
│  Agent: Haiku                                               │
│  Depends on: Task 01, Task 05                               │
└─────────────────────────────────────────────────────────────┘
```

---

## Task Summary

| ID | Task | Agent | Tokens (Est.) | Dependencies |
|----|------|-------|---------------|--------------|
| 01 | JWT Package | Sonnet | ~12K | None |
| 02 | Auth Callout Module | Opus | ~23K | 01, 05 |
| 03 | PWA Event Subject Alignment | Sonnet | ~13K | None |
| 04 | Request/Reply Handlers | Sonnet | ~18K | 03 |
| 05 | Config & Environment | Haiku | ~6K | 01 |
| 06 | Magic Link Command Support | Haiku | ~5K | 01, 05 |

---

## Agent Selection Rationale

### Opus (Task 02)
- Complex architectural decisions
- New module creation with multiple components
- NATS Auth Callout protocol implementation
- Permission matrix logic

### Sonnet (Tasks 01, 03, 04)
- Moderate complexity implementations
- Following established patterns
- Service layer additions
- DTO creation and mapping

### Haiku (Tasks 05, 06)
- Configuration additions
- Simple handler implementations
- Straightforward integrations

---

## Files Created

```
.claude/tasks/
├── 00-task-index.md           # This file
├── 01-jwt-package.md          # JWT generation/validation
├── 02-auth-callout-module.md  # NATS auth callout
├── 03-pwa-event-subjects.md   # Event subject alignment
├── 04-request-reply-handlers.md # Initial data handlers
├── 05-config-environment.md   # Configuration updates
└── 06-magic-link-command-support.md # Magic link service
```

---

## New Package Structure (After All Tasks)

```
frolf-bot/
├── pkg/
│   ├── jwt/                    # Task 01
│   │   ├── jwt.go
│   │   ├── claims.go
│   │   └── errors.go
│   └── pwa/                    # Task 04
│       ├── dto.go
│       └── mappers.go
├── app/modules/
│   ├── authcallout/            # Task 02
│   │   ├── module.go
│   │   ├── application/
│   │   │   ├── interface.go
│   │   │   └── service.go
│   │   └── infrastructure/
│   │       ├── handlers/
│   │       │   └── auth_handler.go
│   │       └── permissions/
│   │           └── builder.go
│   ├── round/                  # Task 04 additions
│   │   └── infrastructure/
│   │       └── handlers/
│   │           └── pwa_handlers.go  # New
│   └── leaderboard/            # Task 04 additions
│       └── infrastructure/
│           └── handlers/
│               └── pwa_handlers.go  # New
└── config/
    └── config.go               # Task 05 updates
```

---

## Checklist Alignment

From BACKEND_INTEGRATION.md:

### Required
- [x] JWT generation → Task 01, 06
- [x] NATS Auth Callout → Task 02
- [x] Auth Callout permissions → Task 02
- [x] Events with guild_id suffix → Task 03

### Optional (Recommended)
- [x] Round list request/reply → Task 04
- [x] Leaderboard snapshot request/reply → Task 04
- [ ] Guild info endpoint → Not scoped (Discord bot provides)

---

## Execution Commands

Run tasks with Claude Code:

```bash
# Phase 1 (parallel)
claude --task .claude/tasks/01-jwt-package.md &
claude --task .claude/tasks/03-pwa-event-subjects.md &

# Phase 2
claude --task .claude/tasks/05-config-environment.md

# Phase 3 (parallel)
claude --task .claude/tasks/02-auth-callout-module.md &
claude --task .claude/tasks/04-request-reply-handlers.md &

# Phase 4
claude --task .claude/tasks/06-magic-link-command-support.md
```
