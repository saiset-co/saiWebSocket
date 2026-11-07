# saiWebSocket

A Go message broker that routes messages to connected WebSocket clients and registered HTTP callbacks. It provides a small text protocol and an HTTP management API.

Part of the [saiSET](https://github.com/saiset-co) backend ecosystem.

## How it works

- WebSocket clients connect to `/ws` and are associated with routing tokens.
- Messages contain a receiver field and a payload, separated by `|`.
- The broker sends the payload to matching WebSocket clients as a JSON string.
- Registered HTTP callbacks receive the payload as a form field named `message`.
- The HTTP API at `/` manages routing tokens and callbacks and submits messages.

Connections, routing tokens and callbacks are held in memory. The service does not implement a persistent queue, replay or delivery acknowledgements.

## Quick start

[go.mod](go.mod) declares Go 1.21. The supplied Dockerfile builds with Go 1.24.

```sh
git clone https://github.com/saiset-co/saiWebSocket.git
cd saiWebSocket
go mod download
```

Create `saiwebsocket.config` in the working directory. This example is for a local demonstration on the loopback interface:

```json
{
  "Host": "127.0.0.1",
  "Port": "8000",
  "Origin": "*",
  "Responseheaders": "*",
  "AllowUnregisteredClients": "yes",
  "AllowUnregisteredHttpClients": "no",
  "RegisteredTokensUrl": "",
  "AutoTLS": "no"
}
```

```sh
go run ./websockets_pro.go
```

The application reads **`saiwebsocket.config`**, not `websockets.conf`. To build a native executable:

```sh
go build -o saiwebsocket ./websockets_pro.go
```

## Browser example

Run this from a local test page that permits a WebSocket connection to the local server:

```js
const socket = new WebSocket("ws://localhost:8000/ws?RegisterToken=demo1");

socket.onopen = () => {
  socket.send("demo1|Hello");
};

socket.onmessage = (event) => {
  console.log(JSON.parse(event.data));
};
```

The example works with `AllowUnregisteredClients: "yes"`. With that option disabled, the token must already be registered.

### Message format

```text
demo1,demo2|Hello
```

The current implementation matches a client's token using a substring search within the receiver field. It does not parse an exact-match recipient list or implement hierarchical group routing. In particular, `group/|Hello` does not automatically address a client registered as `group/item1`. Choose non-overlapping routing tokens when evaluating this protocol.

### Service messages

| Message | Behavior |
| --- | --- |
| `RegisterToken:<token>` | Change the routing token of an established connection |
| `Echo:<message>` | Return a JSON-encoded echo response |

Changing a token on an established connection does not revalidate it against the registered-token list. Routing tokens should therefore not be treated as a complete authentication or authorization boundary.

## HTTP management API

Send form-encoded requests to the server root. Method names and parameter names are case-sensitive.

| `method` | Additional fields | Purpose |
| --- | --- | --- |
| `broadcast` | `message` | Route a receiver-field/payload message |
| `get_clients` | None | List connected WebSocket clients |
| `registerToken` | `token` | Enable a routing token |
| `unregisterToken` | `token` | Mark a routing token as disabled |
| `registeredTokenList` | None | List routing-token states |
| `registerHttpClient` | `token`, `endpoint` | Register an HTTP callback |
| `httpClientsList` | None | List registered callbacks |

Register a token:

```sh
curl -X POST http://localhost:8000/ \
  --data-urlencode 'method=registerToken' \
  --data-urlencode 'token=demo1'
```

Send a message:

```sh
curl -X POST http://localhost:8000/ \
  --data-urlencode 'method=broadcast' \
  --data-urlencode 'message=demo1|Hello'
```

Disabling a token controls subsequent registration checks; it does not disconnect an existing WebSocket client.

## Configuration

| Field | Meaning |
| --- | --- |
| `Host`, `Port` | Listen address and port |
| `AllowUnregisteredClients` | Allow WebSocket connections without a registered token when set to `"yes"` |
| `AllowUnregisteredHttpClients` | Allow callback registration without a registered token when set to `"yes"` |
| `RegisteredTokensUrl` | Optional URL loaded at startup, returning a JSON map of token names to booleans |
| `Crtfile`, `Keyfile` | Certificate and private-key paths for TLS |
| `AutoTLS` | Generate a self-signed certificate when set to `"yes"` and no certificate file is supplied |
| `Origin`, `Responseheaders` | Legacy settings; see deployment notes below |

Example response from `RegisteredTokensUrl`:

```json
{"demo1": true, "demo2": true}
```

### Docker configuration

```sh
cp .env.example .env
```

The [config.template](config.template) uses `HOST` and `PORT`, while the example environment and Compose port mapping use `SERVER_HOST` and `SERVER_PORT`. Add the template variables to `.env` explicitly:

```dotenv
HOST=0.0.0.0
PORT=8090
```

Keep `PORT` aligned with `SERVER_PORT`. Choose the TLS mode for your deployment before starting; the example environment enables self-signed TLS.

```sh
docker compose up -d --build
docker compose logs -f
```

The example Compose mapping publishes host port 8088.

## Deployment notes

The HTTP management API has no built-in authentication. Use this component within a trusted network or behind an authenticated gateway.

The WebSocket origin check compares the configured value to the request URL, rather than implementing a standard Origin-header allowlist. HTTP CORS headers are hard-coded, and `Responseheaders` does not configure them.

The implementation also shares in-memory maps between concurrent handlers without synchronization. Concurrency hardening and load testing are required before using it as a production messaging boundary.

Source: [websockets_pro.go](websockets_pro.go). API examples are also available in the [Postman collection](Websockets.postman_collection.json).

