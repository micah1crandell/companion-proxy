# Companion Proxy

## Overview
Companion Proxy is a command-line utility and HTTP server that allows you to create, manage, and trigger HTTP actions. It provides a simple way to interact with remote APIs using configurable actions, which can be added, edited, deleted, or triggered via the command line or a web interface.

## Features
- Start a local HTTP server to manage actions.
- Add, edit, delete, and list HTTP actions.
- Trigger actions via CLI or HTTP API.
- Store logs of executed actions.
- Lightweight and easy to use.

---

## Quick Start

### Run Without Building a Binary
To quickly start the server without building a binary, run:

```sh
 go run main.go server -port=8080
```

This will start the HTTP server on port `8080`. You can specify a different port if needed.

---

### Build and Run as a Binary
If you prefer to build and run Companion Proxy as a standalone binary, follow these steps:

1. Build the binary:
   ```sh
   go build -o companion-proxy main.go
   ```
2. Run the binary:
   ```sh
   ./companion-proxy server -port=8080
   ```

Now you can use the CLI to interact with the server:

```sh
./companion-proxy add -name "MyAction" -url "http://example.com/api" -method POST
```

---

### Install Binary for System-Wide Use
If you want to run `companion-proxy` from any directory without specifying `./` or the full path, move the binary into a directory included in your system’s `PATH`:

```sh
mv companion-proxy /usr/local/bin/
```

Now you can run commands globally, such as:

```sh
companion-proxy server -port=8080
```

---

## CLI Usage

### Start Server
```sh
companion-proxy server -port=8080
```

Start the HTTP server on the specified port (default: `8080`).

---

### Add an Action
```sh
companion-proxy add -name "MyAction" -url "http://example.com/api" -method POST -header "Authorization:Bearer token" -body '{"key": "value"}'
```

Creates a new HTTP action.

**Options:**
- `-name`: Unique name for the action (required)
- `-url`: Target URL (required)
- `-method`: HTTP method (default: `POST`)
- `-header`: Header in "Key:Value" format (optional)
- `-body`: Request body (optional)

---

### Edit an Action
```sh
companion-proxy edit -id <actionID> -name "UpdatedAction" -url "http://example.com/updated"
```

Edits an existing action.

**Options:**
- `-id`: Action ID (required)
- `-name`: New name (optional)
- `-url`: New URL (optional)
- `-method`: New HTTP method (optional)
- `-header`: Header in "Key:Value" format (optional)
- `-body`: New request body (optional)

---

### Delete an Action
```sh
companion-proxy delete -id <actionID>
```

Deletes an existing action.

**Options:**
- `-id`: Action ID (required)

---

### List Actions
```sh
companion-proxy list
```

Displays all available actions.

---

### Trigger an Action
```sh
companion-proxy trigger -name "MyAction"
```

Triggers an action by name or ID.

**Options:**
- `-id`: Action ID (optional)
- `-name`: Action name (optional)

---

### View Logs
```sh
companion-proxy logs
```

Displays logs of executed actions.

---

## API Endpoints
If you are running the server, you can interact with the API:

- **List Actions:** `GET /actions`
- **Get Action:** `GET /actions/{id}`
- **Create Action:** `POST /actions`
- **Edit Action:** `PUT /actions/{id}`
- **Delete Action:** `DELETE /actions/{id}`
- **Trigger Action:** `GET /trigger/{name}`
- **Get Logs:** `GET /logs`

---

## Notes
- Actions are persisted in a JSON file (`companion_proxy_data.json`).
- Headers should be provided in "Key:Value" format.

---

## License
MIT License

