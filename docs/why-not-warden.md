# Architecture Decision Record (ADR): Excluding Octa Warden from Docker Pipeline

**Date:** January 30, 2026
**Status:** Implemented / Archived
**Component:** `rust/warden` (Rust)

## 1. Context
The project utilizes a high-performance **Go** backend with **SQLite** as the persistence layer. To ensure long-term data integrity and detect potential file corruption, a CLI tool named **Octa Warden** was developed using **Rust**.

The original architectural plan was to deploy Warden as a "sidecar" container within the Docker Compose network, executing periodic integrity checks via Cron.

## 2. The Problem
During the integration phase, several critical bottlenecks were identified that outweighed the immediate benefits of the tool:

* **Excessive Build Times:** While the Go API builds in seconds (utilizing Alpine), the Rust compilation process—including dependency fetching—extended the CI/CD pipeline duration to **20-40 minutes** under certain network conditions.
* **Resource Exhaustion:** The compilation of the Rust binary within the Docker context caused significant I/O and RAM overhead, leading to Docker daemon instability (`RPC error: EOF` / OOM kills) on development environments.
* **Complexity vs. Value:** For the current scale of the application, maintaining a heavy compilation step for a weekly maintenance script was deemed "over-engineering."

## 3. The Decision
**Octa Warden has been removed from the active Docker deployment pipeline.**

Instead of relying on an external heavy-duty checker, we have optimized the system stability at the **Application Level**:

1.  **Concurrency Control (Semaphore):** Implemented a traffic-control mechanism (`MaxConcurrentDBOps = 10`) within the Go application. This queues write requests in memory, preventing SQLite `database is locked` errors at the source.
2.  **WAL Mode:** The database is configured to use Write-Ahead Logging (WAL), significantly improving concurrency and reducing the risk of corruption during high load.

## 4. Current Status
* **Source Code:** The Rust source code for Warden is preserved in `rust/warden` as a reference for high-performance file I/O operations and for potential future use in large-scale data recovery scenarios.
* **Deployment:** The `warden` service in `docker-compose.yaml` has been removed out to prioritize development velocity and rapid deployment.

IMPORTANT: and `Dockerfile.warden` is removed