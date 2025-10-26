# 🏗️ Architecture Overview

## 📁 File Structure

```
Subasta-gRPC/
│
├── cmd/
│   ├── server/main.go           ← gRPC Server (Port 50051)
│   ├── client/main.go           ← CLI Client (testing)
│   └── webserver/main.go        ← HTTP Server (Port 8080)
│
├── web/
│   ├── templates/
│   │   └── index.html           ← HTML Structure
│   └── static/
│       ├── css/
│       │   └── styles.css       ← Visual Styles
│       └── js/
│           └── auction.js       ← Business Logic
│
├── api/proto/v1/
│   └── auction.proto            ← gRPC Definitions
│
└── pkg/auction/
    ├── auction.pb.go            ← Generated gRPC Code
    └── auction_grpc.pb.go       ← Generated gRPC Code
```

---

## 🔄 Request Flow

### User Places a Bid

```
┌─────────────┐
│   Browser   │  User types "150" and clicks "Place Bid"
│  (Client A) │
└──────┬──────┘
       │
       │ 1. User Input
       │    - Product: "Laptop"
       │    - Amount: 150
       ▼
┌─────────────────────┐
│   auction.js        │  JavaScript handles:
│   (Frontend Logic)  │  - Validation
│                     │  - Typing detection
│                     │  - State management
└──────┬──────────────┘
       │
       │ 2. HTTP POST /auction.AuctionService/PlaceBid
       │    Content-Type: application/json
       │    Body: {"buyer":"Alice","product":"Laptop","amount":150}
       ▼
┌─────────────────────┐
│  webserver/main.go  │  HTTP → gRPC conversion:
│  (Port 8080)        │  - Parse JSON
│                     │  - Create gRPC request
│                     │  - Add CORS headers
└──────┬──────────────┘
       │
       │ 3. gRPC Call: PlaceBid()
       │    Protocol: Protocol Buffers
       │    Transport: HTTP/2
       ▼
┌─────────────────────┐
│  server/main.go     │  Business logic:
│  (Port 50051)       │  - Validate bid
│                     │  - Check product exists
│                     │  - Verify amount > current
│                     │  - Update price
│                     │  - Store bid
└──────┬──────────────┘
       │
       │ 4. Response: Success + New Price
       │    {"success":true,"message":"Bid accepted","current_price":150}
       ▼
┌─────────────────────┐
│  webserver/main.go  │  gRPC → JSON conversion
└──────┬──────────────┘
       │
       │ 5. JSON Response
       ▼
┌─────────────────────┐
│   auction.js        │  Update UI:
│                     │  - Clear input
│                     │  - Add to history
│                     │  - Refresh catalog
│                     │  - Show success
└──────┬──────────────┘
       │
       │ 6. DOM Update
       ▼
┌─────────────┐
│   Browser   │  User sees:
│  (Client A) │  - "Bid accepted!"
└─────────────┘  - Updated price: $150
                 - Bid in history
```

---

## 🔄 Auto-Refresh Flow (Multi-Client)

### Two Users: Alice (typing) and Bob (browsing)

```
TIME: 0 seconds
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  Status: Browsing          │  Status: Browsing           │
│  Typing: false             │  Typing: false              │
│  Last Price: $100          │  Last Price: $100           │
└──────────────────────────────────────────────────────────┘

TIME: 5 seconds (Auto-refresh triggers)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ✅ Refreshes catalog      │  ✅ Refreshes catalog       │
│  → No changes              │  → No changes               │
│  Last Price: $100          │  Last Price: $100           │
└──────────────────────────────────────────────────────────┘

TIME: 7 seconds (Alice starts typing)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  Status: TYPING "1"        │  Status: Browsing           │
│  Typing: true 🟡           │  Typing: false 🟢           │
│  Refresh: PAUSED           │  Refresh: Active            │
└──────────────────────────────────────────────────────────┘

TIME: 8 seconds (Alice continues typing)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  Status: TYPING "150"      │  Status: Browsing           │
│  Typing: true 🟡           │  Typing: false 🟢           │
│  Refresh: PAUSED           │  Refresh: Active            │
└──────────────────────────────────────────────────────────┘

TIME: 10 seconds (Auto-refresh trigger - but Alice typing!)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ❌ SKIPS refresh          │  ✅ Refreshes catalog       │
│  (preserves "150")         │  → Still no changes         │
│  Typing: true 🟡           │  Last Price: $100           │
└──────────────────────────────────────────────────────────┘

TIME: 11 seconds (Alice places bid)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ✅ Bid submitted: $150    │  Status: Browsing           │
│  ✅ Force refresh          │  Typing: false 🟢           │
│  → New Price: $150 ✨      │  Waiting for next refresh   │
│  Typing: false 🟢          │                             │
└──────────────────────────────────────────────────────────┘

TIME: 15 seconds (Auto-refresh trigger)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ✅ Refreshes catalog      │  ✅ Refreshes catalog       │
│  → Price: $150             │  → NEW: Price: $150 ✨      │
│  (already knows)           │  (sees Alice's bid)         │
└──────────────────────────────────────────────────────────┘

TIME: 17 seconds (Bob starts typing)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  Status: Browsing          │  Status: TYPING "175"       │
│  Typing: false 🟢          │  Typing: true 🟡            │
│  Refresh: Active           │  Refresh: PAUSED            │
└──────────────────────────────────────────────────────────┘

TIME: 20 seconds (Auto-refresh trigger)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ✅ Refreshes catalog      │  ❌ SKIPS refresh           │
│  → Price: $150             │  (preserves "175")          │
│  (no change)               │  Typing: true 🟡            │
└──────────────────────────────────────────────────────────┘

TIME: 22 seconds (Bob places bid)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  Status: Browsing          │  ✅ Bid submitted: $175     │
│  Waiting for refresh       │  ✅ Force refresh           │
│                            │  → New Price: $175 ✨       │
└──────────────────────────────────────────────────────────┘

TIME: 25 seconds (Auto-refresh trigger)
┌──────────────────────────────────────────────────────────┐
│  Client A (Alice)          │  Client B (Bob)             │
│  ─────────────────          │  ─────────────              │
│  ✅ Refreshes catalog      │  ✅ Refreshes catalog       │
│  → NEW: Price: $175 ✨     │  → Price: $175              │
│  (sees Bob's bid)          │  (already knows)            │
└──────────────────────────────────────────────────────────┘

RESULT:
✅ Alice could type without interruption
✅ Bob could type without interruption
✅ Both saw each other's bids
✅ No collision
✅ No lost input
✅ Real-time updates
```

---

## 🧠 State Management

### Per-Client State (auction.js)

```javascript
Client State:
├── currentUser: "Alice"
├── isTyping: false
├── typingTimer: <timeout_id>
├── refreshInterval: <interval_id>
├── lastCatalogHash: "abc123..."
└── savedInputs: {
    "bid-Laptop": {
        value: "150",
        selectionStart: 3,
        selectionEnd: 3
    },
    "bid-Phone": {
        value: "",
        selectionStart: 0,
        selectionEnd: 0
    }
}
```

### Server State (server/main.go)

```go
Server State (shared across all clients):
├── users: map[string]string {
│   "Alice": "Alice",
│   "Bob": "Bob"
│}
├── products: map[string]*ProductInfo {
│   "Laptop": {
│       Seller: "John",
│       Product: "Laptop",
│       InitialPrice: 100.0,
│       CurrentPrice: 175.0  ← Updated by both clients
│   }
│}
└── bids: map[string]*BidInfo {
    "LaptopAlice": {Buyer: "Alice", Amount: 150.0},
    "LaptopBob": {Buyer: "Bob", Amount: 175.0}
}
```

---

## 🔐 Data Flow Layers

```
┌─────────────────────────────────────────────┐
│         PRESENTATION LAYER                  │
│  (What the user sees)                       │
│                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │  HTML    │  │   CSS    │  │   DOM    │ │
│  │ Structure│  │  Styles  │  │ Elements │ │
│  └──────────┘  └──────────┘  └──────────┘ │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│         APPLICATION LAYER                   │
│  (Business logic)                           │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │        auction.js                    │  │
│  │  - Event handling                    │  │
│  │  - State management                  │  │
│  │  - Typing detection                  │  │
│  │  - Input preservation                │  │
│  │  - Change detection                  │  │
│  └──────────────────────────────────────┘  │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│         TRANSPORT LAYER                     │
│  (Protocol conversion)                      │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │      webserver/main.go               │  │
│  │  - HTTP ↔ gRPC conversion            │  │
│  │  - JSON ↔ Protobuf                   │  │
│  │  - CORS handling                     │  │
│  │  - Static file serving               │  │
│  └──────────────────────────────────────┘  │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│         SERVICE LAYER                       │
│  (Core business logic)                      │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │       server/main.go                 │  │
│  │  - User management                   │  │
│  │  - Product management                │  │
│  │  - Bid validation                    │  │
│  │  - Price updates                     │  │
│  │  - Data storage                      │  │
│  └──────────────────────────────────────┘  │
└───────────────────┬─────────────────────────┘
                    │
┌───────────────────▼─────────────────────────┐
│         DATA LAYER                          │
│  (Storage - currently in-memory)            │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │  In-Memory Maps (future: Database)   │  │
│  │  - users    map[string]string        │  │
│  │  - products map[string]*ProductInfo  │  │
│  │  - bids     map[string]*BidInfo      │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

---

## 🎯 Key Design Patterns

### 1. Separation of Concerns
- HTML: Structure only
- CSS: Presentation only
- JS: Behavior only

### 2. Observer Pattern
- Auto-refresh observes time intervals
- Typing detection observes user input
- State changes trigger UI updates

### 3. State Management
- Client-side state (per user)
- Server-side state (shared)
- No state conflicts

### 4. Defensive Programming
- XSS protection (HTML escaping)
- Input validation
- Error handling
- Null checks
