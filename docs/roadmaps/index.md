# Ligo Roadmaps

This directory contains roadmaps and proposals for the Ligo framework.

## Documents

### [NestJS Feature Parity](./nestjs-parity.md)
What Ligo has adopted from NestJS and what is planned or intentionally different.
- Full list of adopted features (core, pipeline, lifecycle hooks)
- Planned separate packages (microservices, ws, graphql, schedule, swagger, database)
- Go-idiomatic differences (builder pattern vs decorators)
- Ecosystem map

### [Package Ecosystem](./ecosystem.md)
How Ligo separates concerns with optional packages for advanced features.
- Core framework scope
- Database integration package (planned)
- Microservices package (planned)
- WebSocket, GraphQL, Scheduling packages
- Comparison with NestJS ecosystem

### [Future Features](./future-features.md)
Proposed features for future releases.
- Microservices (in progress)
- WebSocket support
- Task scheduling (cron jobs)
- GraphQL support
- OpenAPI/Swagger integration
- Database integration
- Caching layer

### [Adapter Proposals](./adapter-proposals.md)
HTTP router adapter implementations and proposals.
- Echo v5 ✅ (Complete)
- Fiber (Proposed)
- Gin (Proposed)
- Chi (Proposed)
- Stdlib (Proposed)

### [0.6 Release](./1.0-release.md)
Completed checklist for the v0.6.0 release.
- Documentation requirements ✅
- Testing coverage goals ✅
- Stability guarantees ✅
- Example applications ✅

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

- **0.5.x**: Feature-complete core, stabilization
- **0.6.x**: Internal restructuring, performance improvements (current)
- **0.7+**: New ecosystem packages (microservices, ws, graphql, schedule)
- **1.0**: Breaking changes only if absolutely necessary
