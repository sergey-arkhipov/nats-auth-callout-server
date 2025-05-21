# NATS Authentication Tool

This project provides a Go-based toolset for NATS authentication, including a CLI tool to generate and test JWT tokens (`generate_token`) and a service for handling NATS authentication (`auth-server`). The project is containerized using Docker, allowing easy deployment and usage in NATS-based applications.

## Features

- **generate_token**: A command-line tool to generate NATS JWT tokens from JSON input and optionally test connectivity to a NATS server.
- **auth-server**: A service that handles NATS authentication, configured via a YAML file.
- **Docker Support**: A single Dockerfile builds both binaries and includes the configuration, with `auth-server` as the default entrypoint.

## Prerequisites

- **Go**: Version 1.23 or later (for local development, optional if using Docker).
- **Docker**: Required to build and run the containerized application.
- **NATS Server**: A running NATS server (e.g., at `nats://localhost:4222`) configured for JWT authentication.
- **Environment Variable**: `NATS_TOKEN_SECRET` must be set for `generate_token` to sign JWT tokens.

## Project Structure

````bash

.
├── auth-server                              # CLI tool to generate and test JWT tokens
│   ├── auth
│   │   └── types.go
│   ├── authkeys
│   │   ├── server_keys.go                   # Server connect
│   │   └── server_keys_test.go
│   ├── authresponse
│   │   ├── handler.go
│   │   └── handler_test.go
│   ├── config
│   │   ├── config.go
│   │   └── config_test.go
│   ├── main.go                             # NATS authentication service
│   ├── tokenvalidation
│   │   ├── tokenvalidation.go
│   │   └── tokenvalidation_test.go
│   └── usersdebug
│       └── users.go
├── config.yml                              # Configuration for auth-server
├── Dockerfile
├── Dockerfile.nats
├── generate_token.go
├── go.mod                                  # Root Go module file (optional)
├── go.sum                                  # Root Go dependencies (optional)
├── nats-server.conf                       # Config for NATS Server with callout
└── README.markdown                        # Project documentation```

## Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd nats-auth-tool
````

### 2. Configure Environment

Set environment variables:

```bash
export NATS_URL="nats://your-nats-server:4222"
export NATS_TOKEN_SECRET="your-secret-key"

```

Build custom NATS server with nats-server.conf:

```bash
docker build --no-cache -t nats-server-with-callback -f Dockerfile.nats .
```

Ensure a NATS server is running (e.g., locally):

```bash
docker run -d -p 4222:4222 -p 8222:8222 -p 9222:9222 --name nats nats-server-with-callback
```

### 3. Build the Docker Image

Build the Docker image using the provided `Dockerfile`:

```bash
docker build -t nats-auth-tool .
```

## Usage

### Running the auth-server (Default)

The `auth-server` is the default service, reading configuration from `/app/config.yml` in the container.

Run the service:

```bash
docker run --rm  -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" -e NATS_URL="$NATS_URL" --name auth-nats nats-auth-tool
```

Configuration:

    NATS_TOKEN_SECRET - secret key for token generation (required)

    NATS_URL - URL of your NATS server (required, e.g. nats://nats-server:4222)

    The service doesn't expose any ports as it's a client to NATS server, not a server itself

### Running generate_token

The `generate_token` binary generates JWT tokens and can test NATS connectivity with the `-test=true` flag.

Generate a token without testing:

```bash
docker run --rm -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" nats-auth-tool generate_token -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"],"deny":[]},"sub":{"allow":["_INBOX.>","TEST.>"],"deny":[]}},"account":"PROD","ttl":600}'
```

Output:

```
Generated token: <jwt-token-string>
```

Generate and test a token (requires NATS server access and running auth-server):

```bash
docker exec -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" auth-nats /app/generate_token -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"],"deny":[]},"sub":{"allow":["_INBOX.>","TEST.>"],"deny":[]}},"account":"PROD","ttl":600}' -server="$NATS_URL" -test=true
```

Output:

```
Generated token: <jwt-token-string>
No Streams defined
```

Use default JSON input:

```bash
docker run --rm -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" nats-auth-tool generate_token
```

Output:

```
No input provided; using default JSON with _INBOX.> permission for NATS request-reply
Generated token: <jwt-token-string>
```

### Configuration

The `auth-server` uses `config.yml` (included in the Docker image at `/app/config.yml`):

```yaml
nats:
  url: "nats://localhost:4222"
  user: "auth"
  pass: "auth"
auth:
  issuer_seed: "SAAGXPXE6IKAIQDYYJGZGNC6SD4PPMF5IZNVXV6UAKYJUFTMS4RWQZXWSI"
  xkey_seed: "SXAKLMX3W2LKKRE5GVBWAOTOMIVJ3YIJQKM3OAW4AKZ23WY4TPTNEJ53TE"
environment: "development"
```

To customize, mount a modified `config.yml`:

```bash
docker run --rm -v $(pwd)/config.yml:/app/config.yml -e NATS_TOKEN_SECRET="your-secret-key" nats-auth-tool
```

The `generate_token` binary uses the `NATS_TOKEN_SECRET` environment variable and optional flags:

- `-input`: JSON string with `user_id`, `permissions`, `account`, and `ttl`.
- `-server`: NATS server URL (default: `nats://localhost:4222`).
- `-test`: Test NATS connection (default: `false`).

## Future Improvements

### GitHub CI/CD for Docker Hub

A GitHub Actions workflow will be added to:

- Build the Docker image on push to the `main` branch.
- Push the image to Docker Hub (e.g., `yourusername/nats-auth-tool`).
- Tag images with the commit SHA and `latest`.

Placeholder for the workflow (to be implemented):

```yaml
name: Build and Push Docker Image
on:
  push:
    branches:
      - main
jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Log in to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: yourusername/nats-auth-tool:latest
```

## Troubleshooting

- **NATS Connection Errors**:
  - Ensure the NATS server is running and accessible (e.g., `nats://localhost:4222`).
  - Verify `NATS_TOKEN_SECRET` matches the server’s secret for `generate_token`.
  - Use `--network=host` or update `config.yml` to point to the correct NATS server host.
- **auth-server Config**:
  - If `auth-server` fails to read `config.yml`, check if it expects a different path or environment variable (e.g., `CONFIG_PATH`).
  - Mount a custom `config.yml` using `-v` if needed.
- **Build Failures**:
  - Ensure `go.mod` and `go.sum` files are present if required.
  - Verify the directory structure matches the expected layout.

## License

MIT License. See [LICENSE](LICENSE) for details.
