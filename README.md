
# Octa

### Performance-Optimized Avatar Generation & Asset Management Engine

Designed for avatar services, profile images, and lightweight asset delivery.


**Octa** is a self-hosted, high-performance service designed for dynamic avatar generation and static asset management. Built with Go and optimized for resource-constrained environments (Low-End VPS), it delivers low-latency response times for thousands of concurrent requests.

By leveraging a custom-tuned SQLite engine with **Write-Ahead Logging (WAL)**, Octa ensures that heavy write operations (uploads) do not block high-frequency read operations (avatar delivery).

---

## Core Features

* **Dynamic Avatar Synthesis:** Generate unique, hash-based avatars using names, emails, or IDs. Supports SVG, PNG, and JPEG formats.
* **Intelligent Asset Mapping:** A robust aliasing system that maps multiple keys to a single physical asset, reducing storage redundancy.
<!-- * **Octa ConsoleUI:** A built-in administrative dashboard for real-time monitoring, asset management, and system health checks. -->
* **Octa ConsoleUI:** A built-in administrative dashboard for real-time monitoring, asset management, and system health checks.

![Octa ConsoleUI](./assets/consoleui.png)


* **Stateless Scaling:** Designed to be horizontally scalable when backed by shared storage or high-performance volumes.
* **Hardened Security:** Built-in rate limiting, CORS management, and secret-based authentication for upload endpoints.

---

## Getting Started

### 1. Docker Deployment (Recommended)

Octa is distributed as a lightweight (20MB) Alpine-based image.

```bash
docker compose up -d --build

```

### 2. Manual Compilation

For local development, use the provided `Makefile` to interact with the **GoCraft** build engine.

```bash
# Run directly from source
make run

# Standard build
make build

# Optimized production build via GoCraft
make craft

```

---

## Detailed Configuration

Octa uses a hierarchical configuration system. It first looks for a `config.yaml` file and then overrides values with **Environment Variables**.

### Environment Variable Mapping

Keys in `config.yaml` map to environment variables using an uppercase, underscore-separated format.
Example: `security.upload_secret` becomes `AVATAR_SECURITY_UPLOAD_SECRET`.

### 1. Server & Application

| Key | ENV Variable | Default | Description |
| --- | --- | --- | --- |
| `app.name` | - | `Octa` | Application name used in headers/logs. |
| `server.port` | `APP_PORT` | `9980` | Port for the HTTP server. |
| `server.env` | `APP_ENV` | `development` | `production` enables strict security validation. |
| `base_url` | - | `auto` | Root URL for generating absolute asset links. |

### 2. Database & Storage

| Key | ENV Variable | Default | Description |
| --- | --- | --- | --- |
| `database.path` | `AVATAR_DATABASE_PATH` | `./data/avatar.db` | Local path to the SQLite file. |
| `database.max_size` | - | `2GB` | Soft limit for database auto-pruning. |
| `database.prune_interval` | - | `5m` | Frequency of the background cleanup worker. |

### 3. Image Processing & Caching

| Key | Default | Description |
| --- | --- | --- |
| `image.default_size` | `360` | Default dimensions for avatars. |
| `image.quality` | `80` | JPEG/WebP compression quality (1-100). |
| `image.max_upload_size` | `5MB` | Maximum allowed size for multipart uploads. |
| `cache.enabled` | `true` | Enables in-memory LRU caching for hot assets. |
| `cache.max_capacity` | `100` | Cache size in MB. |

### 4. Security & Rate Limiting

| Key | ENV Variable | Description |
| --- | --- | --- |
| `security.upload_secret` | `AVATAR_SECURITY_UPLOAD_SECRET` | Required header for POST /upload. |
| `consoleui.user.username` | `ADMIN_DASHBOARD_USERNAME` | Admin login credential. |
| `consoleui.user.password` | `ADMIN_DASHBOARD_PASSWORD` | Admin login credential. |
| `security.rate_limit.requests` | - | Allowed requests per window. |
| `security.rate_limit.window` | - | Time window (e.g., `1s`, `1m`). |

---

## API Reference

### Dynamic Avatar Endpoint

Generates a unique visual representation for a given key.

**`GET /avatar/{key}`**

| Query Param | Type | Default | Example |
| --- | --- | --- | --- |
| `theme` | string | `gradient` | `theme=gradient/auto` |
| `bg` | hex | random | `bg=f7b1b1` |
| `color` | hex | `dynamic` | `color=000000` |
| `size` | int | `360` | `size=512` |
| `rounded` | bool/int(1-100) | `false` | `rounded=true`, `rounded=75` |

### Asset Management

Upload and retrieve stored assets.

* **Upload:** `POST /upload` (Requires `X-Upload-Secret` header)
* **Retrieve:** `GET /u/{alias_or_id}`

---

## The Octa Ecosystem

The project includes an internal suite of tools for infrastructure maintenance and performance verification:

* **GoCraft Build Engine:** A dedicated build script that handles cross-compilation, version injection, and binary stripping.
* **GoBench (Benchmark):** A RuGost-based load tester designed to simulate high-concurrency write/read scenarios. Verify your system via `make bench-go`.
* **Octa-Pulse (Benchmark):** A Rust-based load tester designed to simulate high-concurrency write/read scenarios. Verify your system via `make bench-rust`.
* **Octa-Warden (Forensic Audit):** A Rust utility that performs deep inspection of the SQLite database to ensure binary(blob) integrity without downtime. Access via `make warden`.

---

## License

MIT License. Developed by [Onur Artan](https://www.google.com/search?q=https://github.com/onurartan).
