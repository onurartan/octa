# ADR: Testing Scope and Prioritization

**Date:** 30.01.2026
**Status:** Accepted

## Context
Octa is designed as a specialized micro-service (Unix philosophy: "Do one thing well") to support a larger ecosystem. It is not the primary product but a supporting utility.

## Decision
**Comprehensive Unit Testing was deprioritized in favor of Integration Benchmarks.**

## Reasoning
1.  **Resource Allocation:** Engineering resources are currently allocated to the core product. Octa required a fast "Time-to-Implementation".
2.  **Complexity Profile:** The service logic is linear (I/O bound). Complex state management—which usually necessitates unit tests—is minimal.
3.  **Validation:**
    * **Benchmarks:** Rust/Go load tools confirmed stability under 20k+ concurrent requests.
    * **Manual QA:** Critical paths (Upload/Resize) were verified manually.

## Future Scope
If business logic complexity increases, Unit Tests will be introduced primarily for the `oc` module.