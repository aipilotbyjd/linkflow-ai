# Architecture Audit & Review

## 1. Executive Summary
The architecture defined in `ARCHITECTURE.md` describes a robust, scalable microservices system using Domain-Driven Design (DDD), Event-Driven Architecture (EDA), and Clean Architecture principles. It is well-suited for a high-scale n8n-like workflow automation platform. However, there are significant gaps between the ambitious architecture and the current implementation status, as well as some specific "n8n-clone" requirements that need more detailed architectural definition.

## 2. Discrepancies Found

### Port Assignments
There is a conflict in port assignments between `ARCHITECTURE.md` and `PROJECT_STATUS.md`.

| Service | Architecture Port | Project Status Port |
| :--- | :--- | :--- |
| **Execution Service** | 8005 | 8003 |
| **Tenant Service** | 8003 | *Not Implemented* |
| **Workflow Service** | 8004 | 8004 (Matches) |
| **Node Service** | 8006 | 8005 |

**Recommendation**: Align the architecture document to match the current implementation (Status), or update the implementation to match the architecture. Given `PROJECT_STATUS.md` reflects reality, `ARCHITECTURE.md` likely needs updating.

### Service Priorities
`PROJECT_STATUS.md` lists **Node Service** and **Schedule Service** as high priority. The Architecture document doesn't specify implementation order, but the "Conclusion" implies a finished state.

### Implementation Gaps
- **CQRS**: Marked as "Partial" in status but key in architecture.
- **Testing**: Architecture describes a "Testing Pyramid" but status says "~0% coverage".
- **Communication**: Architecture specifies gRPC for internal comms; Status says it's "Missing" and a medium priority.

## 3. n8n-Specific Architectural Gaps

### Execution Sandbox (Critical)
The `Executor Service` (Port 8007) is responsible for "Sandboxed environments". For an n8n clone allowing custom JavaScript/Python code in nodes, the security implications are massive.
*   **Current Def**: "Sandboxed environments, Resource limits".
*   **Missing Detail**: specific technology choice (e.g., Firecracker, gVisor, V8 Isolates, or simple Docker containers). Docker alone is often considered insufficient for multi-tenant code execution security.

### Large Payload Handling
Workflow automation often involves passing large data sets (JSON, binary files) between nodes.
*   **Current Arch**: Relies on Kafka for events and PostgreSQL for state.
*   **Risk**: Passing multi-megabyte payloads through Kafka or storing them in Postgres JSONB columns for every step will cause performance bottlenecks.
*   **Recommendation**: Hybrid storage pattern. Use Object Storage (MinIO/S3) for payloads > 1MB, passing only references (Claim Check Pattern) via Kafka/DB.

## 4. Conclusion & Roadmap
To "conclude" the architecture, we must acknowledge the journey from the current state to the target state. The architecture document should serve as the "North Star".

**Proposed Action**:
1.  Update `ARCHITECTURE.md` to fix port conflicts.
2.  Add a "Data Handling Strategy" section for large payloads.
3.  Expand "Executor Service" to specify Sandbox Strategy options.
4.  Update the "Conclusion" of `ARCHITECTURE.md` to include a "Roadmap to Production" that bridges the gap between the status and the vision.
