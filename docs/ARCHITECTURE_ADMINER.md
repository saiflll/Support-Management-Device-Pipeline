# Adminer Security Architecture

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                           USER BROWSER                               │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │                           │
                    ▼                           ▼
        ┌───────────────────┐       ┌───────────────────┐
        │   OTA Dashboard   │       │ Forwarder Dashboard│
        │   Port: 9999      │       │   Port: 8888       │
        │                   │       │                    │
        │ ┌───────────────┐ │       │ ┌────────────────┐ │
        │ │ Login Page    │ │       │ │ Login Page     │ │
        │ │ - Request Code│ │       │ │ - Request Code │ │
        │ │ - Input Code  │ │       │ │ - Input Code   │ │
        │ └───────┬───────┘ │       │ └────────┬───────┘ │
        │         │         │       │          │         │
        │         ▼         │       │          ▼         │
        │ ┌───────────────┐ │       │ ┌────────────────┐ │
        │ │Session Store  │ │       │ │ Session Store  │ │
        │ │(Memory)       │◄┼───────┼─┤ (Memory)       │ │
        │ └───────────────┘ │       │ └────────────────┘ │
        │         │         │       │          │         │
        │         │         │       │          │         │
        │    [Database]     │       │     [Database]     │
        │    Button Click   │       │     Button Click   │
        └─────────┬─────────┘       └──────────┬─────────┘
                  │                            │
                  └────────────┬───────────────┘
                               │
                               ▼
                  ┌────────────────────────┐
                  │   Adminer Proxy        │
                  │   Port: 8080           │
                  │                        │
                  │ ┌────────────────────┐ │
                  │ │ Session Validator  │ │
                  │ │                    │ │
                  │ │ Check Cookie:      │ │
                  │ │ - authenticated?   │ │
                  │ │ - expired?         │ │
                  │ └─────────┬──────────┘ │
                  │           │            │
                  │      ┌────┴────┐       │
                  │      │         │       │
                  │   Valid?   Invalid     │
                  │      │         │       │
                  └──────┼─────────┼───────┘
                         │         │
                         │         └──────► Redirect to
                         │                  /login (OTA)
                         ▼
              ┌──────────────────┐
              │    Adminer       │
              │  (Internal Only) │
              │                  │
              │  No External     │
              │  Port Exposed    │
              └────────┬─────────┘
                       │
                       ▼
              ┌──────────────────┐
              │   PostgreSQL     │
              │   Database       │
              │                  │
              │  Port: 5432      │
              │  (Internal)      │
              └──────────────────┘
```

## Authentication Flow

```
┌──────┐                                    ┌──────────┐
│ USER │                                    │ Telegram │
└───┬──┘                                    └────┬─────┘
    │                                            │
    │ 1. Open /login                             │
    ├──────────────────────►┌─────────┐          │
    │                       │   OTA   │          │
    │                       │    or   │          │
    │                       │Forwarder│          │
    │                       └────┬────┘          │
    │                            │               │
    │ 2. Click "Minta Kode"      │               │
    ├───────────────────────────►│               │
    │                            │               │
    │                            │ 3. Generate   │
    │                            │    6-digit    │
    │                            │    code       │
    │                            │               │
    │                            │ 4. Send code  │
    │                            ├──────────────►│
    │                            │               │
    │                            │               │ 5. Receive
    │◄───────────────────────────┼───────────────┤    code
    │                            │               │
    │ 6. Input code              │               │
    ├───────────────────────────►│               │
    │                            │               │
    │                            │ 7. Validate   │
    │                            │    code       │
    │                            │               │
    │ 8. Set session cookie      │               │
    │◄───────────────────────────┤               │
    │    authenticated=true      │               │
    │    expires=24h             │               │
    │                            │               │
    │ 9. Click "Database"        │               │
    ├───────────────────────────►│               │
    │                            │               │
    │ 10. Redirect to /adminer   │               │
    │◄───────────────────────────┤               │
    │                            │               │
    │                            │               │
    │ 11. Request with cookie    │               │
    ├──────────────────────►┌────┴─────┐         │
    │                       │ Adminer  │         │
    │                       │  Proxy   │         │
    │                       └────┬─────┘         │
    │                            │               │
    │                            │ 12. Validate  │
    │                            │     session   │
    │                            │               │
    │ 13. Proxy to Adminer       │               │
    │◄───────────────────────────┤               │
    │                            │               │
    │ 14. Adminer UI             │               │
    │◄──────────────────────┐    │               │
    │                       │    │               │
    │                    Adminer │               │
    │                   Container│               │
    │                       │    │               │
    └───────────────────────┘    │               │
                                 │               │
                                 ▼               ▼
```

## Security Layers

```
┌─────────────────────────────────────────────────────────┐
│                    Security Layer 1                      │
│              Telegram 2FA Verification                   │
│  - 6-digit random code                                   │
│  - 5-minute expiration                                   │
│  - One-time use                                          │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Security Layer 2                      │
│                 Session Management                       │
│  - HTTPOnly cookies (prevent XSS)                        │
│  - SameSite=Lax (prevent CSRF)                          │
│  - 24-hour expiration                                    │
│  - Secure random session ID                              │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Security Layer 3                      │
│                  Proxy Validation                        │
│  - Check session on every request                        │
│  - Auto-redirect if invalid                              │
│  - No direct database access                             │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│                    Security Layer 4                      │
│               Network Isolation                          │
│  - Adminer: No external ports                            │
│  - PostgreSQL: Internal network only                     │
│  - Docker network isolation                              │
└─────────────────────────────────────────────────────────┘
```

## Data Flow

### Successful Access:
```
User Login → Telegram Code → Session Created → Click Database
    → Adminer Proxy → Validate Session → Forward to Adminer
    → Access Database ✅
```

### Failed Access (No Session):
```
User → Direct Access http://localhost:8080 → Adminer Proxy
    → No Session Found → Redirect to /login ❌
```

### Failed Access (Expired Session):
```
User → Click Database → Adminer Proxy → Session Expired
    → Redirect to /login ❌
```

## Component Responsibilities

| Component | Responsibility | Security Role |
|-----------|---------------|---------------|
| **OTA/Forwarder** | User authentication | Generate & validate codes |
| **Telegram Bot** | Code delivery | Secure out-of-band channel |
| **Session Store** | Session management | Store auth state |
| **Adminer Proxy** | Access control | Validate every request |
| **Adminer** | Database UI | Isolated, no direct access |
| **PostgreSQL** | Data storage | Network isolated |

## Network Topology

```
┌─────────────────────────────────────────────────────────────┐
│                      External Network                        │
│                                                              │
│  User Browser ──► Port 9999 (OTA)                           │
│              ──► Port 8888 (Forwarder)                      │
│              ──► Port 8080 (Adminer Proxy)                  │
│                                                              │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────┴──────────────────────────────────┐
│                   Docker Network: iot-net                    │
│                   Subnet: 192.168.200.0/24                   │
│                                                              │
│  ┌──────────┐  ┌────────────┐  ┌──────────────┐            │
│  │   OTA    │  │ Forwarder  │  │Adminer Proxy │            │
│  │ :9999    │  │   :8888    │  │    :8080     │            │
│  └────┬─────┘  └─────┬──────┘  └──────┬───────┘            │
│       │              │                 │                     │
│       └──────────────┴─────────────────┘                     │
│                      │                                       │
│                      ▼                                       │
│              ┌──────────────┐                                │
│              │   Adminer    │                                │
│              │ (No External │                                │
│              │    Ports)    │                                │
│              └──────┬───────┘                                │
│                     │                                        │
│                     ▼                                        │
│              ┌──────────────┐                                │
│              │  PostgreSQL  │                                │
│              │    :5432     │                                │
│              │  (Internal)  │                                │
│              └──────────────┘                                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Session Lifecycle

```
┌─────────────┐
│ User Login  │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│ Telegram Code Sent  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Code Validated      │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────────────┐
│ Session Created             │
│ - session_id: random        │
│ - authenticated: true       │
│ - expires: now + 24h        │
└──────┬──────────────────────┘
       │
       ▼
┌─────────────────────────────┐
│ User Activity               │
│ (Access Adminer, etc)       │
└──────┬──────────────────────┘
       │
       ├──► Every Request: Session Validated
       │
       ▼
┌─────────────────────────────┐
│ Session Expiry Check        │
└──────┬──────────────────────┘
       │
       ├──► Still Valid ──► Continue
       │
       └──► Expired ──► Redirect to Login
```
