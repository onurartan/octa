# Octa-Warden

> **⚠️ Status: Standby / On-Demand**
> This tool is currently decoupled from the active CI/CD pipeline to prioritize build velocity.
> It is designed to be run **manually** or via **scheduled maintenance jobs** rather than as a runtime sidecar.
> *See [ADR-001: Architectural Decision](../docs/why-not-warden.md) for details.*

**Octa-Warden** is a specialized, high-performance database integrity auditor designed for the Octa System. Written in **Rust**, it operates as a forensic tool to ensure the structural and data integrity of assets stored within SQLite.

## 🏗 Architecture & Design Philosophy

While the core Octa server (Go) handles high-concurrency traffic, **Octa-Warden** serves as the system's "Deep Scan Unit." It is completely decoupled from the main server logic, adhering to the **Separation of Concerns (SoC)** principle.

### Why Rust?
We chose Rust for this component over Go or Python for specific architectural reasons:
1.  **Zero-Cost Abstractions:** Rust allows for high-level logic with low-level memory control, essential for scanning gigabytes of BLOB data without garbage collection pauses.
2.  **Safety & Stability:** The borrow checker ensures memory safety without a runtime cost. If a database row contains malformed data, Rust's strict type system catches it immediately (Fail-Safe).
3.  **SIMD Image Decoding:** Warden uses SIMD-accelerated libraries to decode image headers in milliseconds, allowing for rapid full-table scans.

---

## Features

* **Non-Destructive Audit:** Operates in `READ_ONLY` mode via WAL (Write-Ahead Logging). It guarantees that the audit process will never lock the database, allowing it to run alongside the live Go server.
* **Deep Inspection:** It validates not just file existence but decodes the BLOB headers in memory to verify they are valid image assets (PNG/JPEG/WebP).
* **Fail-Safe Iteration:** If a specific row is corrupted, Warden logs the specific error and continues scanning the rest of the dataset.

---

## 🛠 Usage

### 1. Manual Audit (CLI)
Warden is compiled as a standalone binary for system administrators to perform spot checks or post-incident recovery.

```bash
# Summon the Warden via Makefile
make warden

```

**Output Example:**

```text
Octa Warden - Database validation tool

→ Loading configuration from: config.yaml
[OK] Database connected. Integrity audit starting...

[DB-ERR] X Schema Mismatch | Reason: InvalidColumnType(1, "data", Text)
[CORRUPT] ! ID: user-123-uuid | Reason: Format error decoding Png

WARDEN AUDIT REPORT
Time Elapsed   : 142.3ms
Assets Scanned : 1500
--------------------------------
Healthy Assets : 1498
Corrupted Blobs: 1
Schema Errors  : 1
--------------------------------
Status         : ATTENTION REQUIRED

```

### 2. Docker Integration Strategy

*Initially designed as a startup sidecar.*

While Warden includes a `Dockerfile` for containerized environments, it is currently **disabled by default** in the main `docker-compose.yaml` to minimize image build times during development.

In a **Production** environment with massive datasets (100GB+), Warden can be enabled as a **CronJob Container** (e.g., running every Sunday at 03:00 AM) to ensure long-term data health without impacting API performance.

---

## Configuration

Warden reads the shared `config.yaml` used by the main application.

```yaml
# config.yaml
database:
  path: "./data/octa.db"

```

## Error Codes

| Code | Type | Description | Action Required |
| --- | --- | --- | --- |
| **[CORRUPT]** | `Asset Error` | The BLOB data cannot be decoded as an image. | The file was likely truncated. Row deletion recommended. |
| **[DB-ERR]** | `Schema Error` | Column data type mismatch. | Manual SQL intervention required. |
