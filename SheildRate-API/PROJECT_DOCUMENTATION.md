# API Rate Limiter Project - Complete Documentation

## 📋 Project Overview

**Project Name:** API Rate Limiter  
**Language:** Go 1.24.0  
**Purpose:** A production-ready API rate limiting service with authentication, client management, and Go language feature demonstrations.

---

## 🏗️ Project Architecture

```
api-rate-limiter/
├── main.go                    # Server entry point & route handlers
├── go.mod                     # Go module definition
├── config/
│   └── config.go             # Configuration variables & settings
├── middleware/
│   ├── ratelimiter.go        # Rate limiting middleware logic
│   └── cors.go               # CORS middleware for cross-origin requests
├── rate-limiter/
│   ├── logic.go              # Core rate limiting algorithm & data structures
│   ├── auth.go               # Authentication & password hashing (bcrypt)
│   ├── json.go               # JSON serialization/deserialization utilities
│   ├── pointer.go            # Pointer demonstration functions
│   ├── auth_test.go          # Authentication unit tests
│   └── json_test.go          # JSON handling unit tests
├── frontend/
│   └── index.html            # Frontend UI (if applicable)
└── lab/
    └── goroutines_demo.go    # Goroutine demonstrations
```

---

## 🔑 Key Components Breakdown

### 1. **Configuration Module** (`config/config.go`)
Centralized configuration management:
- **MaxRequests (int):** Maximum allowed requests per time window (default: 5)
- **WindowDuration (time.Duration):** Time window for rate limiting (default: 10 seconds)

**Feature:** Global variables for easy configuration management

---

### 2. **Rate Limiter Core** (`rate-limiter/logic.go`)

#### Data Structures:

**Metadata Struct (Embedded)**
```go
type Metadata struct {
    ClientID string
}
```
- Used for embedding in Client struct (composition pattern)

**Client Struct**
```go
type Client struct {
    Metadata         // Embedded struct
    RequestCount int
    WindowStart  time.Time    // When current window started
    LastSeen     time.Time    // Last activity timestamp
    RequestLog   []time.Time  // Request history
    PasswordHash string       // Bcrypt password hash
}
```

**RateLimiter Struct** (Main service)
```go
type RateLimiter struct {
    Clients      map[string]*Client  // Client registry
    Mutex        sync.RWMutex        // Thread-safe map access
    allowedCount atomic.Uint64       // Atomic counter for allowed requests
    blockedCount atomic.Uint64       // Atomic counter for blocked requests
    stopCleanup  chan struct{}       // Channel to signal cleanup stop
    cleanupDone  chan struct{}       // Channel to signal cleanup finished
}
```

#### Key Features:

1. **Constructor: `NewRateLimiter()`**
   - Uses `make()` to create maps and channels
   - Spawns background goroutine for cleanup worker

2. **Rate Limiting Logic: `Allow(clientID, maxRequests, window)`**
   - **Validation:** Checks if clientID is empty (returns error if true)
   - **Lock Management:** Uses RWMutex for thread-safe access
   - **Window Reset:** Resets client state if window expired
   - **Counting:** Tracks RequestCount and RequestLog
   - **Return:** Boolean (allowed/blocked) + error

3. **Cleanup Worker: `startCleanupWorker()`**
   - **Runs in background:** Goroutine with ticker
   - **Cleans inactive clients:** Removes clients without activity for 5 minutes
   - **Graceful shutdown:** Responds to stopCleanup channel
   - **Interval:** Checks every 30 seconds by default

4. **Statistics: `Stats()`**
   - Returns allowed/blocked request counts
   - Returns active client count

5. **Shutdown: `Shutdown()`**
   - Gracefully stops cleanup worker
   - Ensures proper goroutine cleanup

---

### 3. **Authentication Module** (`rate-limiter/auth.go`)

#### Features:

1. **Password Hashing: `HashPassword(password)`**
   - Uses bcrypt with DefaultCost
   - Securely hashes passwords with salt
   - Returns hash string and error

2. **Password Verification: `CheckPassword(hash, password)`**
   - Compares provided password with stored hash
   - Uses bcrypt's constant-time comparison
   - Prevents timing attacks

3. **Client Registration: `RegisterClient(clientID, password)`**
   - Creates new client entry with hashed password
   - Initializes metadata and timestamps
   - Thread-safe with RWMutex locking

4. **Client Authentication: `Authenticate(clientID, password)`**
   - Verifies client credentials
   - Updates LastSeen timestamp on success
   - Returns error if client doesn't exist or password mismatches

---

### 4. **JSON Handling** (`rate-limiter/json.go`)

#### Features:

1. **Serialization: `ToJSON(client)`**
   - Converts Client struct to JSON bytes
   - Uses json.Marshal()

2. **Deserialization: `FromJSON(data)`**
   - Converts JSON bytes back to Client struct
   - Uses json.Unmarshal()

**Demonstrates:** JSON marshaling with struct tags for API communication

---

### 5. **Pointer Operations** (`rate-limiter/pointer.go`)

#### Features:

1. **Call by Value: `IncrementByValue(count int)`**
   - Takes a copy of the value
   - Changes don't affect original variable
   - Demonstrates pass-by-value semantics

2. **Call by Pointer: `IncrementByPointer(count *int)`**
   - Takes address of the value
   - Changes affect the original variable
   - Demonstrates pass-by-reference semantics

**Demonstrates:** Difference between value and reference semantics in Go

---

### 6. **Rate Limit Middleware** (`middleware/ratelimiter.go`)

#### Workflow:

```
Request arrives
    ↓
Extract Client ID (from header or use remote address)
    ↓
Check Authorization header (Bearer token required)
    ↓
Validate token format
    ↓
Call rl.Allow() to check rate limit
    ↓
If allowed: Pass to next handler
If blocked: Return 429 Too Many Requests
If error: Return 400 Bad Request
```

#### Key Functions:

1. **RateLimitMiddleware(rl, next)**
   - Wraps HTTP handler with rate limiting
   - Extracts client ID from X-Client-ID header
   - Validates Bearer token from Authorization header
   - Calls Allow() method to enforce limit
   - Returns appropriate HTTP status codes

2. **respondWithError(w, message, statusCode)**
   - Helper to return JSON error responses
   - Sets Content-Type header
   - Anonymous struct for JSON marshaling

#### HTTP Status Codes Used:
- **200 OK:** Request allowed
- **400 Bad Request:** Missing/invalid client ID
- **401 Unauthorized:** Missing or invalid authentication token
- **429 Too Many Requests:** Rate limit exceeded

---

### 7. **CORS Middleware** (`middleware/cors.go`)

#### Features:

1. **CORSMiddleware(next)**
   - Allows all origins (*)
   - Allows headers: Content-Type, X-Client-ID, Authorization
   - Allows methods: GET, POST, OPTIONS
   - Handles preflight requests (OPTIONS)

---

### 8. **Main Server** (`main.go`)

#### Startup Process:

```
Initialize Rate Limiter
    ↓
Run demonstrations:
  - Pointer semantics (call by value vs reference)
  - JSON serialization/deserialization
  - bcrypt authentication
    ↓
Register multiple test clients
    ↓
Set up HTTP routes:
  - GET / → Protected API endpoint (rate-limited + authenticated)
  - POST /auth/login → Authentication endpoint
    ↓
Apply middleware stack:
  - CORS → Rate Limit → Handler
    ↓
Start server on port 8080
```

#### API Endpoints:

**1. POST /auth/login**
- **Purpose:** Authenticate client and receive token
- **Request Body:**
  ```json
  {
    "clientID": "client1",
    "password": "mypassword"
  }
  ```
- **Success Response (200):**
  ```json
  {
    "token": "demo_token_client1",
    "clientID": "client1",
    "message": "Authentication successful",
    "expiresAt": "3600s"
  }
  ```
- **Error Response (400):** Invalid request format
- **Error Response (401):** Invalid credentials

**2. GET / (Protected Endpoint)**
- **Purpose:** Example protected API endpoint
- **Required Headers:**
  - `X-Client-ID: <client_id>`
  - `Authorization: Bearer <token>`
- **Rate Limit:** 5 requests per 10 seconds
- **Success Response (200):**
  ```json
  {
    "message": "API request successful",
    "status": "ok"
  }
  ```
- **Error Response (401):** Missing/invalid token
- **Error Response (429):** Rate limit exceeded

---

## 🔄 Request Flow Diagram

```
Client Request
    ↓
CORS Middleware Check
    ├─ If OPTIONS → Return 200 OK
    ├─ Add CORS headers
    ↓
For /auth/login:
    ├─ Parse JSON (clientID, password)
    ├─ Call Authenticate()
    ├─ Verify with bcrypt
    ├─ Return token
    ↓
For / endpoint:
    ├─ Verify Authorization header
    ├─ Call RateLimiter.Allow()
    ├─ Check rate limit status
    ├─ If allowed → Process request
    ├─ If blocked → Return 429
    ├─ Return response
```

---

## 🧵 Concurrency Features

### 1. **Mutexes (Synchronization)**
- **RWMutex in RateLimiter:** Allows multiple concurrent readers, single writer
- **Lock:** Used when modifying client state
- **RLock:** Used for read-only operations

### 2. **Atomic Operations**
- **allowedCount:** Thread-safe counter for allowed requests
- **blockedCount:** Thread-safe counter for blocked requests
- **No locking needed:** Atomic operations are lock-free

### 3. **Goroutines**
- **Cleanup Worker:** Background goroutine that removes inactive clients
- **Ticker:** Regular intervals for cleanup checks
- **Channel Communication:** stopCleanup and cleanupDone channels

### 4. **Channel Patterns**
- **Signal channels:** struct{} channels for signaling
- **Buffering:** Unbuffered channels for synchronization

---

## 🔐 Security Features

### 1. **Password Security**
- Bcrypt hashing with salting
- DefaultCost for security/performance balance
- Constant-time comparison to prevent timing attacks

### 2. **Authentication**
- Bearer token validation
- Client ID verification
- Password-based authentication

### 3. **Rate Limiting**
- Per-client rate limits
- Time-window based (sliding window)
- Prevents brute force and denial of service

### 4. **CORS**
- Controlled cross-origin access
- Specific allowed headers and methods

---

## 📊 Data Flow Examples

### Example 1: Client Registration
```
RegisterClient("client1", "mypassword")
    ↓
HashPassword("mypassword") → bcrypt hash
    ↓
Create Client struct with:
    - ClientID: "client1"
    - PasswordHash: <bcrypt_hash>
    - WindowStart: now
    - LastSeen: now
    ↓
Store in RateLimiter.Clients map
```

### Example 2: Rate Limiting Check
```
Allow("client1", 5, 10*time.Second)
    ↓
Check if client exists
    ├─ If new: Create with count=1, return true
    ├─ If existing: Check window expired
    │   ├─ If expired: Reset count to 1
    │   ├─ If not expired:
    │       ├─ If count >= max: Add to blocked, return false
    │       ├─ If count < max: Increment, add to allowed, return true
```

### Example 3: Token Generation
```
User requests /auth/login with credentials
    ↓
Authenticate(clientID, password)
    ├─ Find client in map
    ├─ CheckPassword(hash, password)
    ├─ bcrypt.CompareHashAndPassword()
    ↓
If match: Generate token (simple format: "demo_token_" + clientID)
    ├─ In production: Use JWT
    ↓
Return token to client
```

---

## 🧪 Testing Infrastructure

### Test Files Present:
1. **auth_test.go** - Authentication tests
2. **json_test.go** - JSON marshaling/unmarshaling tests

---

## 🚀 Current Capabilities

✅ **Implemented Features:**
- Rate limiting per client
- Time-window based limiting with reset
- Client authentication with bcrypt
- Bearer token validation
- RESTful API endpoints
- JSON request/response handling
- CORS support
- Background cleanup of inactive clients
- Atomic counters for statistics
- Thread-safe concurrent access
- Graceful server shutdown
- Comprehensive error handling

---

## 🔧 Configuration Summary

| Config | Default | Location | Purpose |
|--------|---------|----------|---------|
| MaxRequests | 5 | `config/config.go` | Max requests per window |
| WindowDuration | 10s | `config/config.go` | Time window for limit |
| Cleanup Interval | 30s | `rate-limiter/logic.go` | How often to clean inactive clients |
| Idle Threshold | 5m | `rate-limiter/logic.go` | Inactivity duration before cleanup |

---

## 📝 Important Notes

1. **Token Generation is Simple:** Currently uses `"demo_token_" + clientID`. In production, should use JWT.
2. **CORS is Permissive:** Allows all origins (*). Should be restricted in production.
3. **Thread Safety:** All shared data structures are protected with mutexes or atomic operations.
4. **Cleanup is Automatic:** Background worker ensures no memory leaks from stale clients.

---

## 🎯 Ready for Modifications

This project is now well-documented and ready for enhancements. You mentioned wanting to make changes - please describe what modifications you'd like to implement!

### Potential Areas for Enhancement:
- JWT token generation and validation
- Database persistence for clients
- More granular rate limit configurations
- Request history analytics
- Admin endpoints for client management
- Distributed rate limiting (Redis)
- Custom rate limiting strategies
