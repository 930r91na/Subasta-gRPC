# Complete Setup Guide 

### Terminal 1: Start gRPC Server
```powershell
cd D:\Users\Georg\VSCode\Subasta-gRPC\Server
go run main.go
```

**Expected output:**
```
Auction server started on port 50051...
```

### Terminal 2: Start Web UI Server
```powershell
cd D:\Users\Georg\VSCode\Subasta-gRPC\WebUI
.\web-server.exe
```

**Expected output:**
```
🚀 Web UI Server started on http://localhost:8080
📂 Serving files from Client directory
```

---

## STEP 3: Open Browser

```
http://localhost:8080/auction.html
```

---

## What Just Happened?

```
┌─────────────────┐
│    Browser      │
│ localhost:8080  │  ← You open this
└────────┬────────┘
         │
         │ HTTP JSON requests
         ▼
┌─────────────────┐
│  Web UI Server  │
│  (WebUI/main.go)│  ← Converts JSON to gRPC
│   Port 8080     │
└────────┬────────┘
         │
         │ gRPC calls
         ▼
┌─────────────────┐
│  gRPC Server    │
│ (Server/main.go)│  ← Your existing server
│   Port 50051    │
└─────────────────┘
```
