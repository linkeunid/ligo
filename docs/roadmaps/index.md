# Ligo Roadmaps

This directory contains roadmaps and proposals for the Ligo framework.

## Documents

### [Package Ecosystem](./ecosystem.md)
How Ligo separates concerns with optional packages for advanced features.
- Core framework scope
- Database integration package (planned)
- Microservices package (planned)
- WebSocket, GraphQL, Scheduling packages
- Comparison with NestJS ecosystem

### [1.0 Release Roadmap](./1.0-release.md)
Timeline and checklist for the 1.0 release.
- Documentation requirements
- Testing coverage goals
- Stability guarantees
- Example applications

### [Future Features](./future-features.md)
Proposed features for post-1.0 releases.
- WebSocket support
- Task scheduling (cron jobs)
- GraphQL support
- gRPC support
- OpenAPI/Swagger integration
- Message queue integrations
- Caching layer
- Distributed tracing

### [Adapter Proposals](./adapter-proposals.md)
HTTP router adapter implementations and proposals.
- Echo v5 ✅ (Complete)
- Fiber (Proposed)
- Gin (Proposed)
- Chi (Proposed)
- Stdlib (Proposed)

---

## Contributing

If you want to work on any of these features:

1. Check if there's an existing issue
2. Open a discussion for new proposals
3. Follow the existing code patterns
4. Implement as a separate package that imports `github.com/linkeunid/ligo`
5. Include tests and documentation

---

## Version Policy

- **0.9.x**: Bug fixes, minor improvements
- **1.0**: Production-ready with stability guarantees
- **1.x**: New features as separate packages (ws, schedule, graphql, etc.)
- **2.0**: Breaking changes only if absolutely necessary
