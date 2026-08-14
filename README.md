# Go Router

A lightweight Router that selects an **enabled and healthy Main Load Balancer** and redirects the client to it.

## Overview

```text
Clients
   |
   v
api.example.com
   |
   v
DNS
   |
   +----> Router 1
   +----> Router 2
   +----> Router 3
              |
              v
       Enabled + Healthy
          Main LB
              |
              v
        Application/API
```

### Single Router

```text
Client
  |
  v
api.example.com
  |
  v
Router
  |
  v
Main LB
```

### Multiple Routers

Multiple Router instances can share the same hostname. DNS distributes clients among the Router instances.

Each Router independently:

1. Reads `mainLBserver.json`.
2. Ignores every `enable: false` Main LB.
3. Checks the health of enabled Main LBs.
4. Selects an enabled and healthy Main LB.
5. Returns a `307 Temporary Redirect`.
6. The client then connects directly to the selected Main LB.

Router instances do not need to communicate with each other.

## API

### `GET /health`

Checks the Router itself.

```text
GET {base_url}/health

200 OK
router ok
```

### `GET /route`

Selects an enabled and healthy Main LB.

```text
GET {base_url}/route
```

Successful response:

```http
HTTP/1.1 307 Temporary Redirect
Location: https://lb2.example.com
```

If no eligible Main LB is available:

```http
HTTP/1.1 503 Service Unavailable
```

## Main LB Configuration

File: `mainLBserver.json`

```json
[
  {
    "name": "main-lb-1",
    "base_url": "https://lb1.example.com",
    "health": "https://lb1.example.com/health",
    "enable": true
  },
  {
    "name": "main-lb-2",
    "base_url": "https://lb2.example.com",
    "health": "https://lb2.example.com/health",
    "enable": true
  },
  {
    "name": "main-lb-3",
    "base_url": "https://lb3.example.com",
    "health": "https://lb3.example.com/health",
    "enable": false
  }
]
```

### Fields

| Field | Meaning |
|---|---|
| `name` | Main LB name |
| `base_url` | Main LB endpoint the client is redirected to |
| `health` | Full URL the Router calls to check this LB's health |
| `enable` | `true` = eligible, `false` = never selected |

A disabled Main LB is never returned, even if its health check succeeds.

## Client Integration

The client calls the Router to obtain a Main LB.

```text
Client
  |
  | GET /route
  v
Router
  |
  | 307 + Location
  v
Selected Main LB
  |
  v
Client uses Main LB directly
```

### Axios

```js
import axios from "axios";

async function getMainLB() {
  const response = await axios.get(
    "https://router.example.com/route",
    {
      maxRedirects: 0,
      validateStatus: status => status === 307
    }
  );

  return response.headers.location;
}

async function callAPI(path) {
  const baseURL = await getMainLB();
  const response = await axios.get(baseURL + path);
  return response.data;
}
```

### Reuse

Normally, the client should not call `/route` before every request.

```text
/route
  ↓
Main LB 2

/login       → Main LB 2
/movies/123  → Main LB 2
/search      → Main LB 2
```

If the selected Main LB fails, request `/route` again.

## Client Failover

```js
async function requestWithFailover(path) {
  let baseURL = await getMainLB();

  try {
    return (await axios.get(baseURL + path)).data;
  } catch {
    baseURL = await getMainLB();
    return (await axios.get(baseURL + path)).data;
  }
}
```

Flow:

```text
Client → Router → Main LB 2
                     X
                     ↓
Client → Router → Main LB 1 → Success
```

Keep retries limited to avoid infinite retry loops.

## Dynamic Documentation Base URL

`docs.html` uses the current browser origin for its displayed Base URL.

```text
Local:
http://localhost:8080

Render:
https://your-app.onrender.com
```

The docs do not require a hardcoded deployment URL.

## Multiple Router Deployment

Example:

```text
                 api.example.com
                        |
                        v
                       DNS
                    /   |   \
                   /    |    \
                  v     v     v
                R1     R2     R3
                 \      |     /
                  \     |    /
                   \    |   /
                    Main LBs
```

Simple DNS round-robin distributes clients across Router instances.

The DNS layer does **not** choose the Main LB. The Router does that.

All Router instances should have the same `mainLBserver.json`.

## Run

```bash
go mod tidy
go run .
```

Open:

```text
http://localhost:8080
```

The root endpoint serves `docs.html`.

## Render

Typical Render settings for a Go service:

```text
Build Command:
go build -o app .

Start Command:
./app
```

The application should listen on Render's `PORT` environment variable.

## Documentation

Full API documentation is available at the root endpoint:

```text
GET /
```


## Free DNS Distribution — Cloudflare

For multiple Routers, **Cloudflare Free DNS** can provide simple DNS round-robin distribution by publishing multiple `A` records with the same hostname. Cloudflare officially documents multiple `A`/`AAAA` records for simple round-robin DNS.

> **Using Render (or another host that only gives you a hostname, e.g. `router1.onrender.com`, not a static IP)?** The `A`-record setup below needs a fixed IPv4 per Router. Skip ahead to [Render Deployments (No Static IP)](#render-deployments-no-static-ip).

> **Important:** Free DNS round-robin is not the same as health-checked load balancing. DNS caching can make distribution uneven, and a failed Router is not automatically removed by simple round-robin. The Router is responsible for checking Main LB health.

### Architecture

```text
100K Clients
     |
     v
api.example.com
     |
     v
Cloudflare DNS
     |
     +----> Router 1
     +----> Router 2
     +----> Router 3
                |
                v
         Enabled + Healthy
            Main LB
```

### Requirements

```text
1. Registered domain
2. Free Cloudflare account
3. Multiple Router instances
4. Public IPv4 address for each Router
5. Same Router configuration on every Router
```

## Render Deployments (No Static IP)

Render (and most PaaS platforms) does not give free services a fixed public IPv4 address. Instead, each deployment gets its own hostname, e.g.:

```text
router1.onrender.com
router2.onrender.com
router3.onrender.com
```

This breaks the `A`-record round-robin approach above, because there is no IP to point multiple `A` records at. It also cannot be worked around with multiple `CNAME` records: DNS only allows **one** `CNAME` per hostname — Cloudflare (and DNS itself) rejects a second `CNAME` record on the same name.

```text
api            A     203.0.113.10   ✔ works (round-robin)
api            A     203.0.113.20   ✔ works (round-robin)

api            CNAME router1.onrender.com   ✔ works alone
api            CNAME router2.onrender.com   ✘ rejected — only one CNAME per name
```

So a single hostname like `api.example.com` cannot transparently round-robin across several `*.onrender.com` Routers using free DNS alone. Two practical options:

### Option A — Give each Router its own subdomain, let the client pick

Point one `CNAME` at each Router:

```text
router1.example.com  CNAME  router1.onrender.com
router2.example.com  CNAME  router2.onrender.com
router3.example.com  CNAME  router3.onrender.com
```

The client keeps a small list of Router URLs instead of relying on DNS to distribute traffic, and applies the same pick/retry logic already used for Main LB failover:

```js
const ROUTERS = [
  "https://router1.example.com",
  "https://router2.example.com",
  "https://router3.example.com",
];

function pickRouter() {
  return ROUTERS[Math.floor(Math.random() * ROUTERS.length)];
}

async function getMainLB() {
  const base = pickRouter();
  const response = await axios.get(`${base}/route`, {
    maxRedirects: 0,
    validateStatus: status => status === 307,
  });
  return response.headers.location;
}
```

This stays entirely on free tiers, but the Router-selection logic moves from DNS into the client.

### Option B — Cloudflare Load Balancing (paid)

Cloudflare's Load Balancing product supports pools of origins addressed **by hostname** (not just IP), with active health checks and automatic failover — this is the direct replacement for round-robin `A` records when your origins are hostnames like `*.onrender.com`. It is a paid add-on, not part of the Free plan.

| Approach | Cost | DNS complexity | Health-aware |
|---|---|---|---|
| Option A: client-side list | Free | Low | No (client retries on failure) |
| Option B: Cloudflare Load Balancing | Paid | Low | Yes |
| `A`-record round robin (above) | Free | Low | No — requires static IPs, not available on Render |

For a small number of Routers, Option A is usually the simplest starting point.

### Step 1 — Add the domain

1. Create/log in to your Cloudflare account.
2. Add your domain.
3. Select the **Free** plan.
4. Review/import the existing DNS records.
5. Cloudflare will provide nameservers for the zone.
6. At your domain registrar, replace the current nameservers with the Cloudflare nameservers.
7. Wait until the domain becomes active in Cloudflare.

Cloudflare's standard full/primary DNS setup is available on the Free plan and requires the domain's authoritative nameservers to point to Cloudflare.

### Step 2 — Deploy the Routers

Example:

```text
Router 1 → 203.0.113.10
Router 2 → 203.0.113.20
Router 3 → 203.0.113.30
```

These are example IPs only. Use the real public IPv4 addresses of your Router servers.

### Step 3 — Create the DNS records

Go to:

```text
Cloudflare Dashboard
→ Your Domain
→ DNS
→ Records
→ Add record
```

Create these records:

| Type | Name | Content | Proxy status | TTL |
|---|---|---|---|---|
| A | `api` | `203.0.113.10` | DNS only | Auto |
| A | `api` | `203.0.113.20` | DNS only | Auto |
| A | `api` | `203.0.113.30` | DNS only | Auto |

All three records have the same name: `api`.

So the public hostname is:

```text
api.example.com
```

Cloudflare supports multiple `A` or `AAAA` records for the same hostname for simple round-robin DNS.

### Step 4 — Keep the Router configuration identical

Every Router should have the same `mainLBserver.json`:

```json
[
  {
    "name": "main-lb-1",
    "base_url": "https://lb1.example.com",
    "health": "https://lb1.example.com/health",
    "enable": true
  },
  {
    "name": "main-lb-2",
    "base_url": "https://lb2.example.com",
    "health": "https://lb2.example.com/health",
    "enable": true
  },
  {
    "name": "main-lb-3",
    "base_url": "https://lb3.example.com",
    "health": "https://lb3.example.com/health",
    "enable": false
  }
]
```

The Router, not DNS, decides which Main LB is healthy and enabled.

### Step 5 — Final request flow

```text
Client
  |
  v
api.example.com
  |
  v
Cloudflare DNS
  |
  +----> Router 1
  |
  +----> Router 2
  |
  +----> Router 3
             |
             v
      Main LB selection
             |
             +----> LB1
             +----> LB2
             +----> LB3
```

### Step 6 — Test DNS

Windows:

```bash
nslookup api.example.com
```

Linux/macOS:

```bash
dig api.example.com
```

You should see Router IPs being returned over time. DNS caching means distribution will not necessarily be perfectly even per request.

### DNS vs Router responsibilities

| Layer | Job |
|---|---|
| Cloudflare DNS | Distribute DNS answers across Router IPs |
| Router | Select enabled + healthy Main LB |
| Main LB | Handle application/API traffic |

### Important limitation

Simple DNS round-robin does **not** provide precise request-level balancing or automatic health-based removal of failed Router instances. Cloudflare notes that more advanced traffic control such as automatic failover and intelligent routing is provided by its Load Balancing service.

For a simple/free setup:

```text
Cloudflare Free DNS
        +
Multiple Routers
        +
Router-side Main LB health checks
```

is the basic architecture.

### Quick Setup Checklist

```text
[ ] Domain added to Cloudflare
[ ] Cloudflare nameservers set at registrar
[ ] Router 1 deployed
[ ] Router 2 deployed
[ ] Router 3 deployed
[ ] Public IPs collected
[ ] 3 A records created with Name: api
[ ] Records set to DNS only
[ ] Same mainLBserver.json on every Router
[ ] DNS tested with nslookup/dig
[ ] api.example.com tested from a client
```
