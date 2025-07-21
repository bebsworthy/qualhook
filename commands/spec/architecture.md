# Architecture Analysis Command

Perform comprehensive codebase analysis to create or update the global architecture documentation.

## Usage
```
/spec:architecture [--new]
```

## Options
- `--new`: Create new architecture.md, replacing any existing one (default: update existing)

## Instructions

You are performing comprehensive analysis of the codebase to document its architecture. This documentation will be reused across all feature specifications to avoid redundant analysis.

### Process

1. **Check Existing Documentation**
   - Check if `.documentation/architecture.md` exists
   - If exists and `--new` flag NOT provided:
     - Load existing architecture.md
     - Focus on updating with new findings
     - Preserve existing accurate information
   - If doesn't exist OR `--new` flag provided:
     - Perform complete analysis from scratch

2. **System Overview Discovery**
   - Identify architecture style by looking for:
     - Multiple service directories (services/, apps/, packages/)
     - Docker/docker-compose files indicating multi-container setup
     - Multiple build configuration files in subdirectories
     - Service mesh configurations (Istio, Linkerd)
     - API gateway configurations
   - Create high-level component inventory

3. **Module/Service Discovery**
   - For monolithic applications:
     - Treat as single module
     - Document complete tech stack in one section
   - For multi-module applications:
     - Scan each service/module directory separately
     - Look for build files in each: package.json, go.mod, pom.xml, etc.
     - Identify different tech stacks per module
     - Map module boundaries and responsibilities

4. **Per-Module Analysis**
   For each discovered module/service:
   - **Technology Stack**: Language, framework, build tools specific to this module
   - **Dependencies**: Module-specific dependencies
   - **Entry Points**: Main files for this module
   - **Structure**: Directory organization within the module
   - **Testing**: Test framework and location for this module
   - **APIs**: What this module exposes

5. **Inter-Service Communication Discovery**
   - Look for API definitions (OpenAPI, Proto files, GraphQL schemas)
   - Identify HTTP clients and service calls
   - Find message queue configurations
   - Detect service discovery patterns
   - Map data flow between services

6. **Shared Infrastructure Analysis**
   - Identify shared databases (which services use which DBs)
   - Find shared caching layers
   - Detect authentication/authorization services
   - Map API gateways or load balancers
   - Identify message brokers or event buses

7. **Data Architecture**
   - Map data storage systems and which services use them
   - Document data flow between services
   - Identify transaction boundaries
   - Find data consistency strategies

8. **Development & Testing**
   - Document how to run the full system locally
   - Find orchestration tools (docker-compose, k8s manifests)
   - Identify different test strategies:
     - Unit tests per service
     - Integration tests across services
     - End-to-end test suites
   - Map CI/CD pipelines for multi-service builds

9. **Deployment Architecture**
   - Identify deployment strategies per service
   - Find infrastructure as code
   - Document service discovery mechanisms
   - Map environment configurations

10. **Security & Performance**
    - Document security boundaries between services
    - Identify service-to-service authentication
    - Find rate limiting and throttling
    - Map caching strategies
    - Identify monitoring and tracing setup

11. **Technical Debt & Issues**
    - Search for TODO/FIXME/HACK comments in each module
    - Identify deprecated services or components
    - Note inconsistencies between services
    - Document known integration issues

### Example Discovery Patterns

Common multi-module structures to look for:

```
# Microservices
services/
├── auth-service/     (Go + gRPC)
├── api-gateway/      (Node.js + Express)
├── user-service/     (Java + Spring Boot)
└── frontend/         (React + TypeScript)

# Monorepo with packages
packages/
├── web-app/          (Next.js)
├── mobile-app/       (React Native)
├── shared-ui/        (Component library)
└── api-server/       (NestJS)

# Backend + Frontend
backend/              (Python + FastAPI)
frontend/             (Vue.js)
mobile/              (Flutter)

# Modular monolith
src/
├── modules/
│   ├── auth/
│   ├── billing/
│   └── reporting/
└── shared/
```

### Output

1. Use the architecture template from `.spec/templates/architecture-template.md`
2. Adapt sections based on what you discover:
   - Single module: Simplify to one module section
   - Multi-module: Fill out all module sections
   - No inter-service communication: Skip that section
3. Save to `.documentation/architecture.md`
4. Log the analysis date and scope at the end of the file
5. Report findings summary: "Architecture analysis complete. Found [X] modules/services with [list technologies]. Documentation saved to .documentation/architecture.md"

### Key Principles

- **Discover, Don't Assume**: Make no assumptions about technology choices
- **Document Reality**: Record what IS, not what SHOULD BE
- **Module Awareness**: Recognize that different parts may use different technologies
- **Communication Focus**: Pay special attention to how components interact
- **Be Comprehensive**: Document both the parts and the whole
- **Stay Neutral**: Don't judge architectural decisions, just document them
- **Focus on Patterns**: Identify recurring patterns within and across modules

### Update Mode (Default)

When updating existing architecture.md:
1. Load current documentation
2. Focus on what has changed:
   - New dependencies
   - New patterns discovered
   - Structural changes
   - New technical debt
3. Update relevant sections while preserving accurate existing content
4. Add update log entry with date and changes

### New Mode

When `--new` flag is used:
1. Perform complete analysis from scratch
2. Replace any existing architecture.md
3. Useful when:
   - Major refactoring has occurred
   - Starting fresh analysis
   - Existing documentation is outdated

### Example Usage

```
/spec:architecture
```
Updates existing architecture documentation (or creates if none exists).

```
/spec:architecture --new
```
Creates fresh architecture documentation from scratch.

## Next Steps

After architecture analysis is complete, features can use `/spec:2_research` to perform lightweight, feature-specific research that references this architecture documentation.