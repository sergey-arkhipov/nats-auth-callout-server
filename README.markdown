# NATS Authentication Tool

This project provides tools for authenticating with a NATS server, including an authentication server (`auth-server`) and a token generator (`generate_token`). It supports JWT-based authentication and user management via a YAML configuration file, designed for integration with NATS in development and production environments.

## Features

- **generate_token**: A command-line tool to generate NATS JWT tokens from JSON input and optionally test connectivity to a NATS server.
- **auth-server**: A service that handles NATS authentication, configured via a YAML file.
- **Docker Support**: A single Dockerfile builds both binaries and includes the configuration, with auth-server as the default entrypoint.
- **users support**: A yaml file with users and permissions

## Prerequisites

- **Docker**: Required to build and run the provided images.
- **NATS Server**: A running NATS server instance (e.g., `nats://localhost:4222`).
- **Environment Variables**: Set the following variables before running commands:

```bash
export NATS_URL="nats://your-nats-server:4222"
export NATS_TOKEN_SECRET="your-secret-key"
```

## Setup

### 1. Build the Custom NATS Server

Build a custom NATS server image using the provided `nats-server.conf`:

```bash
docker build --no-cache -t nats-server-with-callback -f Dockerfile.nats .
```

### 2. Run the NATS Server

Start the NATS server locally with the custom image:

```bash
docker run -d -p 4222:4222 -p 8222:8222 -p 9222:9222 --name nats-server nats-server-with-callback
```

### 3. Build the Authentication Tool Image

Build the Docker image for the authentication tool:

```bash
docker build -t nats-auth-tool .
```

## Usage

### Running the Authentication Server

The `auth-server` is the primary service, reading its configuration from `/app/config.yml` inside the container. It connects to a NATS server as a client and does not expose any ports.

Run the authentication server:

```bash
docker run --rm -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" -e NATS_URL="$NATS_URL" --name auth-server nats-auth-tool
```

**Required Environment Variables**:

- `NATS_TOKEN_SECRET`: Secret key for token generation.
- `NATS_URL`: URL of the NATS server (e.g., `nats://nats-server:4222`).

### Generating JWT Tokens

The `generate_token` binary generates JWT tokens for NATS authentication. It supports optional connectivity testing with the `-test=true` flag.

#### Generate a Token Without Testing

```bash
docker run --rm -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" nats-auth-tool generate_token -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"],"deny":[]},"sub":{"allow":["_INBOX.>","TEST.>"],"deny":[]}},"account":"TEST","ttl":600}'
```

**Output**:

```
Generated token: <jwt-token-string>
```

#### Generate and Test a Token

Requires a running `auth-server` and NATS server:

```bash
docker exec -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" auth-server /app/generate_token -test=true -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"],"deny":[]},"sub":{"allow":["_INBOX.>","TEST.>"],"deny":[]}},"account":"TEST","ttl":600}' -server="$NATS_URL"
```

**Output**:

```
Generated token: <jwt-token-string>
No Streams defined
```

#### Use Default Input

If no input is provided, a default JSON with `_INBOX.>` permission is used:

```bash
docker run --rm -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" nats-auth-tool generate_token
```

**Output**:

```
No input provided; using default JSON with _INBOX.> permission for NATS request-reply
Generated token: <jwt-token-string>
```

### Running with Docker Compose

For local development, use Docker Compose to manage services:

```bash
docker compose up
```

#### Generate a Token with Docker Compose

```bash
docker compose run --rm token-generator -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"]},"sub":{"allow":["_INBOX.>","TEST.>"]}},"account":"TEST","ttl":600}'
```

**Output**:

```
Generated token: <jwt-token-string>
```

#### Generate and Test a Token with Docker Compose

```bash
docker compose run --rm token-generator -test=true -input '{"user_id":"bob","permissions":{"pub":{"allow":["$JS.API.>"]},"sub":{"allow":["_INBOX.>","TEST.>"]}},"account":"TEST","ttl":600}' -server="$NATS_URL"
```

**Output**:

```
Generated token: <jwt-token-string>
No Streams defined
```

## Configuration

### Authentication Server

The `auth-server` uses `/app/config.yml` inside the Docker image. Default configuration:

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

### Token Generator

The `generate_token` binary uses the following options:

- `-input`: JSON string specifying `user_id`, `permissions`, `account`, and `ttl`.
- `-server`: NATS server URL (default: `nats://localhost:4222`).
- `-test`: Enable connectivity testing (default: `false`).
- Environment variable `NATS_TOKEN_SECRET` is required.

### User Management

The `users.yaml` file defines users and their permissions for debugging or fallback authentication. Mount it to the container:

```bash
docker run --rm -v $(pwd)/users.yaml:/app/users.yaml -e NATS_TOKEN_SECRET="$NATS_TOKEN_SECRET" nats-auth-tool
```

An empty `users.yaml` disables username/password authentication. Example `users.yaml`:

```yaml
sys:
  Pass: sys
  Account: SYS
alice:
  Pass: alice
  Account: DEVELOPMENT
  Permissions:
    pub:
      allow:
        - $JS.API.STREAM.LIST
    sub:
      allow:
        - _INBOX.>
        - TEST.test
```

## Future Improvements

### GitHub CI/CD for Docker Hub

A GitHub Actions workflow is planned to:

- Build the Docker image on push to the `main` branch.
- Push the image to Docker Hub (e.g., `yourusername/nats-auth-tool`).
- Tag images with the commit SHA and `latest`.

Placeholder workflow:

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

- **NATS Connection Issues**:

  - Verify the NATS server is running and accessible at the specified `NATS_URL`.
  - Ensure `NATS_TOKEN_SECRET` matches the server's secret for `generate_token`.
  - Use `--network=host` or update `config.yml` to point to the correct NATS server.

- **Configuration Errors**:

  - If `auth-server` cannot read `config.yml`, verify the file path or mount a custom file with `-v`.
  - Check environment variables like `NATS_TOKEN_SECRET` for correctness.

- **Build Issues**:
  - Ensure `go.mod` and `go.sum` are present if required.
  - Confirm the directory structure aligns with the expected layout.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
