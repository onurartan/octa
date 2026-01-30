This documentation provides a comprehensive guide to the configuration system of **Octa**. Octa utilizes a hierarchical configuration model powered by [Viper](https://www.google.com/search?q=https://github.com/spf13/viper), ensuring flexibility across development, staging, and production environments.

---

## Configuration Hierarchy

Octa loads settings in the following order of precedence (highest to lowest):

1. **Environment Variables** (e.g., `AVATAR_SECURITY_UPLOAD_SECRET`)
2. **`config.yaml`** (Local configuration file)
3. **Default Values** (Hardcoded fallback values in `setDefaults()`)

### Environment Variable Mapping

For environment variables, Octa uses an uppercase naming convention where nested keys are separated by underscores.

* **YAML:** `server.port`  **ENV:** `APP_PORT`
* **YAML:** `security.upload_secret`  **ENV:** `AVATAR_SECURITY_UPLOAD_SECRET`

---

## 1. Application Settings (`app`)

Identifies the instance and manages the entry point behavior.

| Key | Type | Description |
| --- | --- | --- |
| `name` | string | The display name of the service (e.g., "Octa"). |
| `version` | string | Current semantic versioning (read-only from config). |
| `landing_page` | bool | If `true`, the root URL `/` serves the Landing UI. |

---

## 2. Server Infrastructure (`server`)

Manages the networking layer and execution context.

| Key | Type | Example | Description |
| --- | --- | --- | --- |
| `port` | int | `9980` | The TCP port Octa listens on. |
| `env` | string | `production` | Execution environment (`development`, `staging`, `production`). |

> **Note:** Setting `env` to `production` enables strict validation, such as requiring a non-default `upload_secret`.

---

## 3. Storage & Database (`database`)

Octa utilizes a custom-tuned SQLite engine for metadata and asset storage.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `path` | string | `./data/avatar.db` | File system path for the SQLite database. |
| `max_size` | string | `2GB` | The soft limit for total data storage before warnings. |
| `prune_interval` | string | `5m` | Frequency of the background cleanup worker (e.g., `1h`, `30m`). |

---

## 4. Image Processing (`image`)

Global constraints for dynamic generation and asset uploads.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `default_size` | int | `360` | The fallback dimension (width/height) for avatars. |
| `quality` | int | `80` | Compression quality for PNG/WebP/JPEG (1-100). |
| `max_upload_size` | string | `5MB` | Maximum file size allowed for the `/upload` endpoint. |
| `max_key_limit` | int | `7` | Maximum number of aliases (keys) mapped to a single image. |

---

## 5. Performance Cache (`cache`)

In-memory caching layer to reduce Disk I/O for "hot" assets.

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | Toggles the in-memory LRU cache. |
| `max_capacity` | int | `100` | Maximum cache size in **MB**. |
| `ttl` | string | `30m` | Time-to-Live for cached items (e.g., `1h`, `15m`). |

---

## 6. Security & Governance (`security`)

Strict controls for authentication, cross-origin resource sharing, and DDoS protection.

### Upload Authentication

* **`upload_secret`**: A unique string required in the `X-Upload-Secret` header for all write operations.
* **Default:** `CHANGE_THIS_IN_ENV`.

### CORS Configuration

* **`cors_origins`**: A whitelist of domains allowed to interact with the API from a browser. Supports wildcards (e.g., `https://**.example.com`).

### Rate Limiting

Octa implements a token-bucket algorithm for rate limiting.

* **`requests`**: Maximum requests allowed per window.
* **`window`**: The timeframe for the limit (e.g., `1s`).
* **`burst`**: Maximum temporary spike allowed above the limit.

---

## 7. Administrative UI (`consoleui`)

Manages the built-in Octa Dashboard for asset and system monitoring.

| Key | Type | Description |
| --- | --- | --- |
| `enabled` | bool | Enables/Disables the dashboard UI. |
| `user.username` | string | Login username (Mapped to `ADMIN_DASHBOARD_USERNAME`). |
| `user.password` | string | Login password (Mapped to `ADMIN_DASHBOARD_PASSWORD`). |

---

## Example `config.yaml`

```yaml
app:
  name: "Octa"
  version: "0.0.1"
  landing_page: true

server:
  port: 9980
  env: "production"

database:
  path: "./data/avatar.db"
  max_size: "5GB"
  prune_interval: "10m"

security:
  upload_secret: "d7f8g9h0j1k2l3m4n5b6v7c8x9"
  rate_limit:
    enabled: true
    requests: 50
    window: "1s"
    burst: 100

```