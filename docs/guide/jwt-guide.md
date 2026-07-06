# JWT Authentication Guide

## Overview

This guide covers using [`github.com/golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) in this project. JWT enables stateless authentication — the server issues a signed token on login, and the client includes it in subsequent requests via the `Authorization: Bearer <token>` header.

---

## Library Setup

```sh
go get -u github.com/golang-jwt/jwt/v5
```

After importing it in your code, run `go mod tidy` to promote it from `// indirect` to a direct dependency.

---

## Core API (`golang-jwt/jwt/v5`)

### Token Creation (HS256)

```go
import "github.com/golang-jwt/jwt/v5"

claims := jwt.MapClaims{
    "sub":   userId,
    "email": email,
    "role":  role,
    "exp":   time.Now().Add(24 * time.Hour).Unix(),
    "iat":   time.Now().Unix(),
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signedToken, err := token.SignedString([]byte(secret))
```

### Custom Claims Struct (recommended over `MapClaims`)

```go
type UserClaims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

claims := UserClaims{
    UserID: userID,
    Email:  email,
    Role:   role,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        Subject:   strconv.Itoa(userID),
    },
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signedToken, err := token.SignedString([]byte(secret))
```

### Token Parsing & Validation

```go
token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return []byte(secret), nil
}, jwt.WithLeeway(30*time.Second))

if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
    // use claims
}
```

**Key parsing options:**
| Option | Purpose |
|---|---|
| `jwt.WithLeeway(d)` | Clock skew tolerance (recommended: 30s) |
| `jwt.WithIssuedAt()` | Validate `iat` claim |
| `jwt.WithIssuer(iss)` | Require specific issuer |
| `jwt.WithAudience(aud)` | Require specific audience |

---

## Suggested Project Architecture

```
HTTP Request
  → internal/middleware/jwt-auth.go (extract & validate token, inject claims into ctx)
    → internal/handler/* (retrieve claims via GetUserClaims(ctx))
      → internal/service/* (business logic)
```

### Token Service (`internal/service/token/token-service.go`)

A dedicated service for all JWT operations:

- `GenerateToken(userID, email, role)` — creates signed JWT
- `ValidateToken(tokenString)` — parses and validates, returns claims
- `KeyFunc(token)` — returns the HMAC secret with signing method whitelist

This is injected into both the auth service (for login) and the middleware (for request validation).

### Middleware (`internal/middleware/jwt-auth.go`)

Standard `net/http` middleware pattern:

```go
func JWTAuth(tokenService *token_service.TokenService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract Authorization header
            // 2. Strip "Bearer " prefix
            // 3. Call tokenService.ValidateToken()
            // 4. On success: store claims in r.Context()
            // 5. On failure: return 401 JSON error
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Register protected routes:

```go
mux.Handle("GET /profile", middleware.JWTAuth(tokenService)(handler))
```

Type-safe context helpers:

```go
func GetUserClaims(ctx context.Context) *token_service.UserClaims
func GetUserID(ctx context.Context) int
func GetUserRole(ctx context.Context) string
```

### Login Flow

1. Client sends `POST /login` with `{"email": "...", "password": "..."}`
2. Auth handler calls `authService.UserLogin(ctx, input)`
3. Auth service queries user by email, compares bcrypt hash
4. On success: calls `tokenService.GenerateToken(id, email, role)`
5. Response: `{"token": "eyJ...", "user": {"id": 1, "name": "...", ...}}`

---

## Best Practices

### 1. Signing Method Whitelist

Prevent **algorithm confusion attacks** — always verify the signing method in the key function:

```go
if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
}
```

Without this check, an attacker could change the header to `"alg": "none"` and bypass verification.

### 2. Always Validate `exp`

The library validates `exp` by default — do not disable it. Use `jwt.WithLeeway(30s)` to handle minor clock skew between services.

### 3. Short-Lived Access Tokens + Refresh Tokens

| Token | Recommended TTL | Where stored |
|---|---|---|
| Access token | 15–60 minutes | Client memory / short-lived |
| Refresh token | 7–30 days | Server DB + client secure storage |

Always pair short-lived access tokens with **refresh tokens** for seamless session extension.

### 4. Secret Management

- **Never** hardcode `JWT_SECRET` in source code
- Store in `.env` for local dev, environment variables / secret manager (Vault, AWS Secrets Manager) for production
- Use at least **32 random bytes** for HS256 (base64-encoded = 44 chars)
- Rotate secrets periodically; during rotation, accept multiple valid keys

### 5. Use `RegisteredClaims`

Embed `jwt.RegisteredClaims` in your custom struct — it provides standard fields (`sub`, `exp`, `iat`, `iss`, `aud`) that the library automatically validates. This is safer and cleaner than `MapClaims`.

### 6. Context Key Safety

Use a typed key (not a raw string) to store claims in `context.Context`:

```go
type contextKey string
const claimsKey contextKey = "user_claims"
```

This prevents collisions with other context values from different packages.

### 7. Environment Variables

Add to `.env`:

```env
JWT_SECRET=your-256-bit-secret-here-min-32-chars
JWT_EXPIRY_HOURS=24
```

---

## GenerateToken Best Practices

### 1. Use a Custom Claims Struct (Not MapClaims)

```go
// ❌ Avoid: type unsafe, no built-in validation
claims := jwt.MapClaims{
    "sub": userID,
    "role": role,
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

// ✅ Recommended: type-safe, auto-validates exp/iat/sub
type UserClaims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
claims := UserClaims{
    UserID: userID,
    Email:  email,
    Role:   role,
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        Subject:   strconv.Itoa(userID),
    },
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
```

`RegisteredClaims` makes `exp` validation automatic — the library checks it during parsing without any extra code.

### 2. Always Set Standard Claims

| Claim | Required | Purpose |
|---|---|---|
| `exp` | **Yes** | Token expiry — prevents unlimited token usage |
| `iat` | Yes | Issued at — useful for revocation checks |
| `sub` | Yes | User identifier — standard way to identify who the token belongs to |
| `jti` | Optional | Unique token ID — used for blacklisting individual tokens |
| `iss` | Optional | Issuer — identifies which service issued the token (multi-tenant) |
| `aud` | Optional | Audience — identifies the intended recipient service |

### 3. Keep the Payload Minimal

Store only what the middleware needs to authorize requests — typically `user_id`, `email`, and `role`. Never store:

- Passwords or password hashes
- API keys or secrets
- Full user objects or large nested data
- Session tokens from other services

JWT headers are only base64-encoded, not encrypted. Anyone who intercepts the token can read the payload.

### 4. GenerateAccessToken Pattern

```go
func (s *TokenService) GenerateAccessToken(userID int, email, role string) (string, error) {
    now := time.Now()
    claims := UserClaims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(s.AccessExpiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            Subject:   strconv.Itoa(userID),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(s.Secret))
    if err != nil {
        return "", fmt.Errorf("failed to sign access token: %w", err)
    }
    return signed, nil
}
```

### 5. GenerateRefreshToken Pattern

Refresh tokens should be **opaque random strings**, not JWTs:

```go
func generateRandomToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", fmt.Errorf("failed to generate random token: %w", err)
    }
    return hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}

func (s *TokenService) GenerateRefreshToken(userID int) (string, error) {
    rawToken, err := generateRandomToken()
    if err != nil {
        return "", err
    }
    _, err = s.DB.Exec(
        `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
         VALUES (?, ?, ?)`,
        userID, hashToken(rawToken), time.Now().Add(s.RefreshExpiry),
    )
    if err != nil {
        return "", fmt.Errorf("failed to store refresh token: %w", err)
    }
    return rawToken, nil
}
```

### 6. Use Environment Config for All Parameters

```go
type TokenConfig struct {
    Secret        string        // loaded from env
    AccessExpiry  time.Duration // loaded from env (default: 15m)
    RefreshExpiry time.Duration // loaded from env (default: 7d)
    Issuer        string        // loaded from env
}
```

Never hardcode expiry or secret values.

---

## ValidateToken Best Practices

### 1. Always Whitelist the Signing Method

The single most important security check — prevents **algorithm confusion attacks**:

```go
func keyFunc(t *jwt.Token) (interface{}, error) {
    // ✅ Whitelist exactly one signing method
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return []byte(secret), nil
}
```

What happens without it: an attacker changes the JWT header to `"alg": "none"` and removes the signature. The library would accept it as valid.

### 2. Use ParseWithClaims Over Parse

```go
// ❌ Parse returns jwt.MapClaims — no type safety, manual casting
token, err := jwt.Parse(tokenString, keyFunc)
claims, ok := token.Claims.(jwt.MapClaims)
role, ok := claims["role"].(string)  // runtime cast, could panic

// ✅ ParseWithClaims returns typed struct — compile-time safety
token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, keyFunc)
claims, ok := token.Claims.(*UserClaims)
role := claims.Role  // direct field access, compile-time checked
```

### 3. Use WithLeeway for Clock Skew

Servers in different data centers may have minor clock differences (up to a few minutes):

```go
// ❌ Strict parsing — valid tokens may be rejected due to clock skew
token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, keyFunc)

// ✅ 30-second leeway — handles NTP sync differences safely
token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, keyFunc,
    jwt.WithLeeway(30*time.Second),
)
```

For higher security, use 0 leeway and ensure all servers are NTP-synchronized.

### 4. Validate Additional Claims Conditionally

```go
token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, keyFunc,
    jwt.WithLeeway(30*time.Second),
    jwt.WithIssuer("helpdesk-api"),           // reject tokens from unknown sources
    jwt.WithAudience("helpdesk-web"),          // reject tokens meant for other services
    jwt.WithIssuedAt(),                        // reject tokens with invalid iat
)
```

Only validate `iss`/`aud` if your architecture uses them — don't add them if they're not set in the token.

### 5. Return Generic Error Messages

Never reveal why a token was rejected — it leaks information to attackers:

```go
// ❌ Bad: leaks specific failure reason
if errors.Is(err, jwt.ErrTokenExpired) {
    return nil, fmt.Errorf("your token expired on %s", claims.ExpiresAt)
}

// ✅ Good: generic message for all auth failures
if err != nil || !token.Valid {
    return nil, fmt.Errorf("invalid or expired token")
}
```

### 6. ValidateAccessToken Pattern (Complete)

```go
func (s *TokenService) ValidateAccessToken(tokenString string) (*UserClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &UserClaims{},
        func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return []byte(s.Secret), nil
        },
        jwt.WithLeeway(30*time.Second),
    )
    if err != nil {
        return nil, fmt.Errorf("invalid or expired token")
    }

    claims, ok := token.Claims.(*UserClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid or expired token")
    }

    return claims, nil
}
```

---

## KeyFunc Best Practices

The `KeyFunc` is the callback passed to `jwt.ParseWithClaims`. It returns the key used to verify the token's signature.

### 1. Structure as a Method on TokenService

```go
func (s *TokenService) KeyFunc(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return []byte(s.Secret), nil
}
```

This keeps the secret access encapsulated.

### 2. Support Key Rotation with KID

When rotating secrets, accept multiple keys via the `kid` (key ID) header:

```go
type TokenService struct {
    CurrentSecret string   // used for signing
    PastSecrets   []string // still valid for verification during rotation
}

func (s *TokenService) KeyFunc(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }

    kid, _ := t.Header["kid"].(string)

    switch kid {
    case "", "current":
        return []byte(s.CurrentSecret), nil
    case "past-1":
        if len(s.PastSecrets) > 0 {
            return []byte(s.PastSecrets[0]), nil
        }
    case "past-2":
        if len(s.PastSecrets) > 1 {
            return []byte(s.PastSecrets[1]), nil
        }
    }
    return nil, fmt.Errorf("unknown key id: %s", kid)
}
```

### 3. Set KID When Signing

```go
func (s *TokenService) GenerateAccessToken(...) (string, error) {
    claims := UserClaims{...}
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    token.Header["kid"] = "current"
    return token.SignedString([]byte(s.CurrentSecret))
}
```

### 4. Rotation Workflow

1. Add new secret to `CurrentSecret`, move old secret to `PastSecrets`
2. Start signing with new secret + `kid: "current"`
3. Existing tokens signed with old secret are still accepted via `kid: "past-1"`
4. After old tokens expire naturally, remove `PastSecrets`

---

## Special Use Cases

### Role-Based Access Control (RBAC)

Check `role` from claims in middleware:

```go
func RequireRole(next http.Handler, roles ...string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        role := GetUserRole(r.Context())
        for _, r := range roles {
            if role == r {
                next.ServeHTTP(w, r)
                return
            }
        }
        lib.JsonErrorResponse(w, http.StatusForbidden, "Insufficient permissions")
    })
}
```

Usage: `RequireRole(handler, "admin", "manager")`

### Refresh Token Implementation Guide

The refresh token pattern solves a key problem: if an access token is stolen, the attacker only has access for a short window (15–60 min). The refresh token is long-lived but stored server-side (DB), so it can be individually revoked.

---

#### Architecture Overview

```
                   ┌───────────────────────┐
                   │   TokenService         │
                   │  ─────────────────    │
                   │  GenerateAccessToken() │  → signed JWT (stateless)
                   │  GenerateRefreshToken()│  → opaque token (stored in DB)
                   │  ValidateRefreshToken()│
                   │  RotateRefreshToken()  │
                   └───────────────────────┘
                            │
            ┌───────────────┴───────────────┐
            ▼                               ▼
   ┌────────────────┐            ┌──────────────────┐
   │  refresh_tokens │            │      JWT          │
   │  table (DB)    │            │  (no server store) │
   │  - token_hash  │            └──────────────────┘
   │  - user_id     │
   │  - expires_at  │
   │  - revoked     │
   └────────────────┘
```

---

#### Why Opaque Refresh Tokens?

Refresh tokens should be **opaque random strings** (not JWTs) because:

| Reason | Detail |
|---|---|
| Revocability | Stored in DB — can be deleted/revoked instantly |
| Rotation | Old token is deleted when a new one is issued — prevents replay |
| No expiry leakage | No embedded `exp` claim that could be ignored by clients |
| Smaller payload | Just a hash in the DB, no claims to verify |

The access token remains a **signed JWT** (stateless, no DB lookup needed for API calls).

---

#### Refresh Tokens Table (Migration SQL)

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    revoked    INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
```

---

#### Token Generation Functions

```go
package token_service

import (
    "crypto/rand"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
    DB            *sql.DB
    Secret        string
    AccessExpiry  time.Duration  // e.g. 15 minutes
    RefreshExpiry time.Duration  // e.g. 7 days
}

type UserClaims struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// ------------------------
// ACCESS TOKEN (signed JWT)
// ------------------------

func (s *TokenService) GenerateAccessToken(userID int, email, role string) (string, error) {
    now := time.Now()
    claims := UserClaims{
        UserID: userID,
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(s.AccessExpiry)),
            IssuedAt:  jwt.NewNumericDate(now),
            Subject:   fmt.Sprintf("%d", userID),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.Secret))
}

// -------------------------
// REFRESH TOKEN (opaque)
// -------------------------

// generateRandomToken creates a cryptographically secure random string.
func generateRandomToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}

// hashToken hashes the raw token for storage so the DB never sees the plain value.
func hashToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}

// GenerateRefreshToken creates an opaque random token, stores its hash in DB,
// and returns the plain token (to be sent to the client).
func (s *TokenService) GenerateRefreshToken(userID int) (string, error) {
    rawToken, err := generateRandomToken()
    if err != nil {
        return "", err
    }

    hashed := hashToken(rawToken)
    expiresAt := time.Now().Add(s.RefreshExpiry)

    _, err = s.DB.Exec(
        `INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
         VALUES (?, ?, ?)`,
        userID, hashed, expiresAt,
    )
    if err != nil {
        return "", err
    }

    return rawToken, nil
}
```

---

#### Token Validation & Rotation

```go
// ValidateRefreshToken checks the raw token against the DB.
// Returns userID if valid, error otherwise.
func (s *TokenService) ValidateRefreshToken(rawToken string) (int, error) {
    hashed := hashToken(rawToken)

    var userID int
    var revoked int
    var expiresAt time.Time

    err := s.DB.QueryRow(
        `SELECT user_id, revoked, expires_at
         FROM refresh_tokens WHERE token_hash = ?`, hashed,
    ).Scan(&userID, &revoked, &expiresAt)

    if err == sql.ErrNoRows {
        return 0, fmt.Errorf("refresh token not found")
    }
    if err != nil {
        return 0, err
    }
    if revoked == 1 {
        return 0, fmt.Errorf("refresh token revoked")
    }
    if time.Now().After(expiresAt) {
        return 0, fmt.Errorf("refresh token expired")
    }

    return userID, nil
}

// RotateRefreshToken validates the old token, deletes it, and issues a new one.
// This prevents replay attacks — if a stolen refresh token is used, the
// legitimate user's token will already be rotated out.
func (s *TokenService) RotateRefreshToken(oldRawToken string) (newRawToken string, userID int, err error) {
    userID, err = s.ValidateRefreshToken(oldRawToken)
    if err != nil {
        return "", 0, err
    }

    hashed := hashToken(oldRawToken)

    // Delete old token (rotation — one-time use)
    _, err = s.DB.Exec(`DELETE FROM refresh_tokens WHERE token_hash = ?`, hashed)
    if err != nil {
        return "", 0, err
    }

    // Issue new token
    newRawToken, err = s.GenerateRefreshToken(userID)
    if err != nil {
        return "", 0, err
    }

    return newRawToken, userID, nil
}

// RevokeAllUserTokens logs out all sessions for a user (e.g. password change).
func (s *TokenService) RevokeAllUserTokens(userID int) error {
    _, err := s.DB.Exec(
        `DELETE FROM refresh_tokens WHERE user_id = ?`, userID,
    )
    return err
}
```

---

#### Full Login Response

```go
// Login endpoint returns both tokens:
{
    "access_token":  "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "a1b2c3d4e5f6...",
    "token_type":    "Bearer",
    "expires_in":    900,
    "user": {
        "id":    1,
        "name":  "John Doe",
        "email": "john@example.com",
        "role":  "employee"
    }
}
```

---

#### Refresh Endpoint Logic

```go
// Handler: POST /refresh
// Body: { "refresh_token": "a1b2c3d4e5f6..." }
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
    var input struct {
        RefreshToken string `json:"refresh_token"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        lib.JsonErrorResponse(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
    if input.RefreshToken == "" {
        lib.JsonErrorResponse(w, http.StatusBadRequest, "refresh_token is required")
        return
    }

    // Rotate: validates old, deletes it, creates new
    newRawToken, userID, err := h.TokenService.RotateRefreshToken(input.RefreshToken)
    if err != nil {
        lib.JsonErrorResponse(w, http.StatusUnauthorized, err.Error())
        return
    }

    // Fetch user details to re-issue access token
    user, err := h.Service.GetUserByID(r.Context(), userID)
    if err != nil {
        lib.JsonErrorResponse(w, http.StatusInternalServerError, "User not found")
        return
    }

    accessToken, err := h.TokenService.GenerateAccessToken(user.ID, user.Email, user.Role)
    if err != nil {
        lib.JsonErrorResponse(w, http.StatusInternalServerError, "Failed to generate token")
        return
    }

    lib.JsonSuccessResponse(w, http.StatusOK, "Token refreshed", map[string]any{
        "access_token":  accessToken,
        "refresh_token": newRawToken,
        "token_type":    "Bearer",
        "expires_in":    900,
    })
}
```

---

#### Logout Endpoints

```go
// POST /logout — revoke specific refresh token
// Body: { "refresh_token": "..." }
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
    var input struct {
        RefreshToken string `json:"refresh_token"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        lib.JsonErrorResponse(w, http.StatusBadRequest, "Invalid request payload")
        return
    }

    // Hash the token and delete from DB
    hashed := hashToken(input.RefreshToken)
    _, err := h.TokenService.DB.Exec(`DELETE FROM refresh_tokens WHERE token_hash = ?`, hashed)
    if err != nil {
        lib.JsonErrorResponse(w, http.StatusInternalServerError, "Logout failed")
        return
    }

    lib.JsonSuccessResponse(w, http.StatusOK, "Logged out successfully", nil)
}

// POST /logout-all — revoke ALL refresh tokens for the authenticated user
func (h *AuthHandler) HandleLogoutAll(w http.ResponseWriter, r *http.Request) {
    claims := middleware.GetUserClaims(r.Context())
    err := h.TokenService.RevokeAllUserTokens(claims.UserID)
    if err != nil {
        lib.JsonErrorResponse(w, http.StatusInternalServerError, "Logout failed")
        return
    }
    lib.JsonSuccessResponse(w, http.StatusOK, "Logged out from all devices", nil)
}
```

---

#### Security Rules

1. **Refresh token rotation** — every time a refresh token is used, the old one is deleted and a new one is issued. If an attacker steals a refresh token and uses it before the legitimate user, the legitimate user's next refresh attempt will fail (the old token is gone), alerting them to the breach.

2. **Store hash, not raw token** — the DB stores `SHA256(token)`. If the DB is leaked, refresh tokens cannot be replayed.

3. **Short access token TTL** — 15 minutes is a good default. Even if a JWT is stolen, the damage window is small.

4. **Refresh token TTL** — 7 days for most apps, 30 days for "remember me" functionality. The longer the TTL, the more important rotation becomes.

5. **Revoke on password change** — call `RevokeAllUserTokens()` when the user changes their password, forcing re-login on all devices.

---

### Token Blacklisting (Immediate Revocation)

For scenarios like "log out all devices" or "password changed":

| Approach | When to use |
|---|---|
| In-memory map (`map[string]bool` + mutex) | Single instance, restart-safe not required |
| Redis with TTL | Distributed systems, auto-cleanup |
| Database table | Persistent, but needs background cleanup |

### Stateless Sessions

JWT is inherently stateless — no server-side session storage needed. Validation only requires the signing key. Ideal for:

- Microservices / distributed systems
- Mobile apps
- Serverless / edge functions

### Multi-Tenant Validation

Different tenants get tokens with distinct `iss` (issuer) claims. Validate in middleware:

```go
if !isValidTenant(claims.Issuer) {
    lib.JsonErrorResponse(w, http.StatusUnauthorized, "Invalid token issuer")
    return
}
```

### Asymmetric Signing (RS256/ES256) for Microservices

Use RS256 when multiple services need to validate tokens but only one service should be able to issue them:

- **Auth service**: holds the private key, issues tokens
- **Other services**: hold only the public key, validate tokens
- No shared secret needed across service boundaries

```go
// Signing (auth service)
privateKey, _ := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
signedToken, _ := token.SignedString(privateKey)

// Validation (any service)
publicKey, _ := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
token, _ := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (interface{}, error) {
    return publicKey, nil
})
```

---

---

## Postman Testing Guide

### Setup Environment Variables

Create a Postman **Environment** with these variables:

| Variable | Initial Value | Description |
|---|---|---|
| `base_url` | `http://localhost:8080` | Server address |
| `access_token` | *(leave empty)* | Will be set by login/refresh |
| `refresh_token` | *(leave empty)* | Will be set by login/refresh |

---

### 1. Register a User

```
POST {{base_url}}/user
Body (raw JSON):
{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securePassword123"
}
```

**Expected response (201):**
```json
{
    "status": 200,
    "message": "Success",
    "data": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com",
        "role": "users"
    }
}
```

---

### 2. Login (Get Tokens)

```
POST {{base_url}}/login
Body (raw JSON):
{
    "email": "john@example.com",
    "password": "securePassword123"
}
```

**Expected response (200):**
```json
{
    "status": 200,
    "message": "Success",
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890",
        "token_type": "Bearer",
        "expires_in": 900,
        "user": {
            "id": 1,
            "name": "John Doe",
            "email": "john@example.com",
            "role": "users"
        }
    }
}
```

**Postman Script (Tests tab)** — auto-save tokens to environment:

```javascript
if (pm.response.code === 200) {
    var data = pm.response.json().data;
    pm.environment.set("access_token", data.access_token);
    pm.environment.set("refresh_token", data.refresh_token);
}
```

---

### 3. Call a Protected Endpoint

```
GET {{base_url}}/profile
Headers:
    Authorization: Bearer {{access_token}}
```

**How to set in Postman:**

| Step | Action |
|---|---|
| 1 | Go to the **Authorization** tab |
| 2 | Select **Type: Bearer Token** |
| 3 | Enter `{{access_token}}` in the token field |

Or manually add a header: `Key: Authorization`, `Value: Bearer {{access_token}}`

**Expected response (200):**
```json
{
    "status": 200,
    "message": "Success",
    "data": {
        "id": 1,
        "name": "John Doe",
        "email": "john@example.com",
        "role": "users"
    }
}
```

If token is missing/expired:
```json
{
    "status": 401,
    "message": "Invalid or expired token"
}
```

---

### 4. Refresh Tokens

Use this when the access token has expired (401 response) but the refresh token is still valid.

```
POST {{base_url}}/refresh
Body (raw JSON):
{
    "refresh_token": "{{refresh_token}}"
}
```

**Postman Script (Tests tab)** — update saved tokens:

```javascript
if (pm.response.code === 200) {
    var data = pm.response.json().data;
    pm.environment.set("access_token", data.access_token);
    pm.environment.set("refresh_token", data.refresh_token);
}
```

**Expected response (200):**
```json
{
    "status": 200,
    "message": "Token refreshed",
    "data": {
        "access_token": "eyJ...new...",
        "refresh_token": "new-opaque-token...",
        "token_type": "Bearer",
        "expires_in": 900
    }
}
```

**Note:** The old refresh token is **invalidated** (rotation). The previous `refresh_token` in your environment will no longer work — the Postman script automatically updates both tokens.

---

### 5. Logout

```
POST {{base_url}}/logout
Body (raw JSON):
{
    "refresh_token": "{{refresh_token}}"
}
```

Expected: `204 No Content` or `200 {"status": 200, "message": "Logged out successfully"}`

---

### 6. Full Flow: Simulate Token Expiry in Postman

To test the refresh flow end-to-end without waiting for natural expiry:

1. Set `JWT_EXPIRY_HOURS=0.1` in your `.env` (6 minutes) for testing
2. Or, manually remove the `access_token` env var and call `/refresh` directly

**Collection-level Pre-request Script** (auto-refresh on 401):

```javascript
// Run before every request
const accessToken = pm.environment.get("access_token");

if (!accessToken) {
    // No token — skip (login will handle this)
    return;
}

// Attach Bearer token (Postman will use the Authorization tab if set,
// but this ensures coverage for all requests)
pm.request.headers.add({
    key: "Authorization",
    value: "Bearer " + accessToken
});
```

**Collection-level Test Script** (auto-refresh when 401):

```javascript
if (pm.response.code === 401) {
    const refreshToken = pm.environment.get("refresh_token");
    if (!refreshToken) {
        console.log("No refresh token available — login required");
        return;
    }

    pm.sendRequest({
        url: pm.environment.get("base_url") + "/refresh",
        method: "POST",
        header: { "Content-Type": "application/json" },
        body: { mode: "raw", raw: JSON.stringify({ refresh_token: refreshToken }) }
    }, function (err, res) {
        if (err || res.code !== 200) {
            console.log("Auto-refresh failed — login required");
            return;
        }
        var data = res.json().data;
        pm.environment.set("access_token", data.access_token);
        pm.environment.set("refresh_token", data.refresh_token);
        console.log("Tokens refreshed automatically");
    });
}
```

---

### Summary Table: Postman Endpoints

| Method | Endpoint | Auth Required | Body |
|---|---|---|---|
| `POST` | `/user` | No | `{ name, email, password }` |
| `POST` | `/login` | No | `{ email, password }` |
| `GET` | `/profile` | Yes (Bearer) | — |
| `POST` | `/refresh` | No | `{ refresh_token }` |
| `POST` | `/logout` | No | `{ refresh_token }` |
| `POST` | `/logout-all` | Yes (Bearer) | — |

---

## Summary

| Concept | Recommendation |
|---|---|
| Signing method | HS256 (single service) / RS256 (microservices) |
| Claims type | Custom struct embedding `jwt.RegisteredClaims` |
| Access token TTL | 15–60 minutes |
| Refresh token TTL | 7–30 days (stored hashed in DB) |
| Clock skew | `jwt.WithLeeway(30s)` |
| Secret source | Env var (`JWT_SECRET`) |
| Storage in context | Typed `contextKey` |
| Middleware pattern | `func(http.Handler) http.Handler` |
| Alg check | Always whitelist expected signing method |
