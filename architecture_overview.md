# SheildRate Full-Stack Architecture

This document outlines the architecture and data-flow of the newly upgraded Go-based Rate Limiter full-stack application.

## High-Level Architecture Diagram
The architecture is structured across three main layers: the static Client-Side Frontend, the HTTP Server Middleware Pipeline, and the Backend Core Logic comprising concurrency limits and asynchronous processes.

```mermaid
graph TD
    %% Styling
    style Frontend fill:#1e293b,stroke:#0ea5e9,stroke-width:2px,color:#f8fafc
    style GoServer fill:#0f172a,stroke:#3b82f6,stroke-width:2px,color:#f8fafc
    style RateLimiterCore fill:#1e1b4b,stroke:#8b5cf6,stroke-width:2px,color:#f8fafc
    style AsyncWorker fill:#312e81,stroke:#6366f1,stroke-width:2px,color:#f8fafc

    subgraph User["External"]
        Client[("Web Browser / UI")]
    end
    
    subgraph Frontend["Frontend Client (HTML/JS/CSS)"]
        UI_Login[Login Form]
        UI_Dash[Analytics Dashboard]
        UI_Requester[API Requester]
        
        UI_Login --> |POST /auth/login<br>Credentials| BackendRouter
        UI_Dash -.-> |Displays Metrics| UI_Requester
        UI_Requester --> |GET /api/status<br>Bearer Token + Auth Headers| BackendRouter
    end

    Client --> |GET / <br> Static Files| StaticServer
    
    subgraph GoServer["Go HTTP Server (:8081)"]
        BackendRouter((Request Router))
        StaticServer["HTTP File Server<br>(Serves ./frontend)"]
        
        BackendRouter --> |Auth Route| AuthHandler
        BackendRouter --> |API Route| MiddlewarePipeline
        
        AuthHandler["Auth Handler<br>(bcrypt validation + JWT generation)"]
        
        subgraph MiddlewarePipeline["Protected Middleware Chain"]
            CORS[CORS Middleware]
            JWT["JWT Validation Middleware<br>(Signature & Expiry Check)"]
            RL_Mid["Rate Limit Middleware<br>(Intersects with Core)"]
            
            CORS --> JWT --> RL_Mid
        end
        
        APIHandler["Core API Endpoint<br>Success JSON Provider"]
        
        RL_Mid --> |If Allowed| APIHandler
        RL_Mid -.-> |If Blocked| Rejects[Returns 429 Too Many Requests]
    end

    subgraph RateLimiterCore["Rate Limiter Engine"]
        RL_Engine{{"In-Memory Token/Map Engine"}}
        Mutex["Concurrency Mutex Locks"]
        RL_Engine --- Mutex
    end
    
    subgraph AsyncWorker["Background Asynchronous Services"]
        LoggerChannel>logChan: chan string]
        LogGoroutine(["Async Logger Goroutine"])
        
        LoggerChannel --> |Consumes messages continuously| LogGoroutine
    end

    %% Connections
    StaticServer -.-> |Returns index.html| Frontend
    AuthHandler -.-> |Returns valid signed string| UI_Login
    RL_Mid <==> |Validates Quotas/Time Windows| RL_Engine
    APIHandler --> |Writes status string| LoggerChannel
    LogGoroutine --> |stdout print formatting| ConsoleOut((Console/Log File))

```

## Component Breakdown

1.  **Frontend Single Page Application (SPA)**:
    *   Served statically straight from Go (`http.FileServer`).
    *   Interacts directly with endpoints via JavaScript `fetch`.
    *   Manages connection tokens through browser `localStorage`.
2.  **Authentication & JWT Pipeline**:
    *   **Login**: The handler validates user passwords via `bcrypt` hashing mechanics.
    *   **Tokens**: Returns HS256 signed JWTs with personalized attributes (client IDs and expiration boundaries).
    *   **Interceptor Middleware**: Ensures any call hitting protected routes (`/api/...`) possesses a legitimate and verified token signature before anything else evaluates.
3.  **Rate Limiter Middleware & Concurrency Engine**:
    *   Extracts connected `X-Client-ID` identities.
    *   Passes identity requests deep into the thread-safe `RateLimiter` structure to tally consumption in predefined sliding or fixed windows. 
    *   Safe concurrent access to rate counts and data objects using Go `sync.Mutex` structures within [logic.go](file:///d:/GitHub/api-rate-limiter/rate-limiter/logic.go) preventing race conditions during immense application traffic.
4.  **Asynchronous Message Offloading (Goroutines/Channels)**:
    *   When the API handles a verified and allowed request, it pushes formatted logging metadata into `logChan := make(chan string, 100)`.
    *   The core request does *not* wait for logging I/O to complete, immediately closing and returning the JSON to the client.
    *   A detached background worker `go asyncLogger()` sequentially processes the backlogged events in memory to write them down securely.
