# Go Router

Lightweight Router for selecting an enabled and healthy Main Load Balancer.

## Features

- Main LB configuration through `mainLBserver.json`
- `enable` flag for each Main LB
- Health check before selection
- Disabled LBs are never selected
- `307 Temporary Redirect` to the selected Main LB
- Client-side failover/retry support
- `/health` endpoint for Router health
- `/` serves `docs.html`

## API

### `GET /health`

Checks the Router itself.

```text
200 OK
router ok
```

### `GET /route`

Selects an `enable: true` Main LB that passes the health check.

Successful response:

```http
HTTP/1.1 307 Temporary Redirect
Location: https://lb2.example.com
```

If no enabled healthy Main LB is available:

```http
HTTP/1.1 503 Service Unavailable
```

## Main LB Configuration

File: `mainLBserver.json`

```json
[
  {
    "name": "main-lb-1",
    "url": "https://lb1.example.com",
    "enable": true
  },
  {
    "name": "main-lb-2",
    "url": "https://lb2.example.com",
    "enable": true
  },
  {
    "name": "main-lb-3",
    "url": "https://lb3.example.com",
    "enable": false
  }
]
```

### `enable`

- `true` → eligible for routing if healthy
- `false` → never selected, even if healthy

## Client Integration

The client calls the Router first:

```text
Client
  ↓
/route
  ↓
Router
  ↓
307 + Location
  ↓
Selected Main LB
```

After receiving the Main LB URL, the client can use that LB directly for normal API requests.

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

## Failover

If the selected Main LB fails, the client can call `/route` again and retry with the newly selected LB.

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

Keep retries limited to avoid infinite retry loops.

## Run

```bash
go mod tidy
go run .
```

Open:

```text
http://localhost:8080
```

The API documentation is served from `docs.html`.
