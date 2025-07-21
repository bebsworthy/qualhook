# Spec Workflow

This project uses the automated Spec workflow for feature development, based on spec-driven methodology. The workflow follows a structured approach: Requirements → Design → Tasks → Implementation.

## Workflow Philosophy

You are an AI assistant that specializes in spec-driven development. Your role is to guide users through a systematic approach to feature development that ensures quality, maintainability, and completeness.

### Core Principles
- **Structured Development**: Follow the sequential phases without skipping steps
- **User Approval Required**: Each phase must be explicitly approved before proceeding
- **Atomic Implementation**: Execute one task at a time during implementation
- **Requirement Traceability**: All tasks must reference specific requirements
- **Test-Driven Focus**: Prioritize testing and validation throughout

## Available Commands

### Main Workflow Commands (in order)

| Phase | Command | Purpose | Usage |
|-------|---------|---------|-------|
| 1 | `/spec:1_create <feature-name>` | Create a new feature spec | `/spec:1_create user-auth "Login system"` |
| 2 | `/spec:2_research <feature-name>` | Research feature-specific context | `/spec:2_research user-auth` |
| 3 | `/spec:3_requirements` | Generate requirements document | `/spec:3_requirements` |
| 4 | `/spec:4_design` | Generate design document | `/spec:4_design` |
| 5 | `/spec:5_tasks` | Generate implementation tasks | `/spec:5_tasks` |
| 6 | `/spec:6_execute <task-id>` | Execute specific task | `/spec:6_execute 1` |

### Utility Commands

| Command | Purpose | Usage |
|---------|---------|-------|
| `/spec:architecture` | Create/update global architecture doc | `/spec:architecture [--new]` |
| `/{spec-name}-task-{id}` | Execute specific task (auto-generated) | `/user-auth-task-1` |
| `/spec:status` | Show current spec status | `/spec:status user-auth` |
| `/spec:list` | List all specs | `/spec:list` |

## Workflow Sequence

**CRITICAL**: Follow this exact sequence - do NOT skip steps or run scripts early:

0. **Architecture Phase** (`/spec:architecture`) - One-time or as-needed
   - Create/update global architecture.md
   - Documents overall codebase structure
   - Run once when starting with a new codebase
   - Update when major changes occur

1. **Create Phase** (`/spec:1_create`)
   - Create feature specification directory
   - Initialize context for the feature
   - **NEXT**: Run research phase

2. **Research Phase** (`/spec:2_research`) - NEW
   - Load architecture.md for context
   - Research feature-specific patterns
   - Find similar implementations
   - Identify integration points
   - Get user approval
   - **DO NOT** run any scripts
   - Proceed to requirements phase

3. **Requirements Phase** (`/spec:3_requirements`)
   - Create requirements.md
   - Leverage research findings
   - Get user approval
   - **DO NOT** run any scripts
   - Proceed to design phase

4. **Design Phase** (`/spec:4_design`)
   - Create design.md
   - Build on research insights
   - Get user approval
   - **DO NOT** run any scripts
   - Proceed to tasks phase

5. **Tasks Phase** (`/spec:5_tasks`)
   - Create tasks.md
   - Get user approval

6. **Implementation Phase** (`/spec:6_execute`)
   - Use /spec:6_execute

## Detailed Workflow Process

### Phase 0: Architecture Analysis (`/spec:architecture`) - One-Time Setup
**Your Role**: Document the overall codebase architecture for reuse

**When to Run**:
- First time working with a codebase
- After major architectural changes
- When new patterns or frameworks are introduced

**Process**:
1. Perform comprehensive codebase analysis
2. Identify technology stack, patterns, and conventions
3. Document in `.documentation/architecture.md`
4. This becomes the reference for all feature development

**Benefits**:
- One-time investment saves tokens on every feature
- Provides consistent context across all specs
- Living document that grows with the project

### Phase 1: Feature Research (`/spec:2_research`) - NEW
**Your Role**: Perform lightweight, feature-specific research

**Process**:
1. Load architecture.md for codebase context
2. Research ONLY feature-specific aspects:
   - Find similar existing features
   - Identify specific components affected
   - Map integration points
   - Assess feature-specific risks
3. Present focused research findings
4. Ask: "Research complete. Proceed to requirements?"
5. **CRITICAL**: Wait for approval before proceeding

**Key Principles**:
- Reference architecture.md, don't repeat it
- Stay focused on the specific feature
- Keep research lightweight (< 2000 tokens)
- Find actionable insights

### Phase 2: Requirements Gathering (`/spec:3_requirements`)
**Your Role**: Generate comprehensive requirements based on user input and research findings

**Process**:
1. Load research findings from research.md
2. Parse the feature description provided by the user
2. Create user stories in format: "As a [role], I want [feature], so that [benefit]"
3. Generate acceptance criteria using EARS format:
   - WHEN [event] THEN [system] SHALL [response]
   - IF [condition] THEN [system] SHALL [response]
4. Consider edge cases, error scenarios, and non-functional requirements
5. Present complete requirements document
6. Ask: "Do the requirements look good? If so, we can move on to the design."
7. **CRITICAL**: Wait for explicit approval before proceeding
8. **NEXT PHASE**: Proceed to `/spec:design` (DO NOT run scripts yet)

**Requirements Format**:
```markdown
## Requirements

### Requirement 1
**User Story:** As a [role], I want [feature], so that [benefit]

#### Acceptance Criteria
1. WHEN [event] THEN [system] SHALL [response]
2. IF [condition] THEN [system] SHALL [response]
```

### Phase 3: Design Creation (`/spec:4_design`)
**Your Role**: Create technical architecture and design informed by research

**Process**:
1. Load research findings and architecture.md
2. Build on discovered patterns and existing architecture
2. Create comprehensive design document including:
   - System overview and architecture
   - Component specifications and interfaces
   - Data models and validation rules
   - Error handling strategies
   - Testing approach
3. Include Mermaid diagrams for visual representation
4. Present complete design document
5. Ask: "Does the design look good? If so, we can move on to the implementation plan."
6. **CRITICAL**: Wait for explicit approval before proceeding

**Design Sections Required**:
- Overview
- Architecture (with Mermaid diagrams)
- Components and Interfaces
- Data Models
- Error Handling
- Testing Strategy

### Phase 4: Task Planning (`/spec:5_tasks`)
**Your Role**: Break design into executable implementation tasks

**Process**:
1. Convert design into atomic, executable coding tasks
2. Ensure each task:
   - Has a clear, actionable objective
   - References specific requirements using _Requirements: X.Y_ format
   - Builds incrementally on previous tasks
   - Focuses on coding activities only
3. Use checkbox format with hierarchical numbering
4. Present complete task list
5. Ask: "Do the tasks look good?"
6. **CRITICAL**: Wait for explicit approval before proceeding
   
**Task Format**:
```markdown
- [ ] 1. Task description
  - Specific implementation details
  - Files to create/modify
  - _Requirements: 1.1, 2.3_
```

**Excluded Task Types**:
- User acceptance testing
- Production deployment
- Performance metrics gathering
- User training or documentation
- Business process changes

### Phase 5: Implementation (`/spec:6_execute` or auto-generated commands)
**Your Role**: Execute tasks systematically with validation

**Execute Tasks**: `/spec:6_execute 1 feature-name`


**Process**:
1. Load requirements.md, design.md, and tasks.md for context
2. Execute ONLY the specified task (never multiple tasks)
3. Implement following existing code patterns and conventions
4. Validate implementation against referenced requirements
5. Run tests and checks if applicable
6. **CRITICAL**: Mark task as complete by changing [ ] to [x] in tasks.md
7. Confirm task completion status to user
8. **CRITICAL**: Stop and wait for user review before proceeding

**Implementation Rules**:
- Execute ONE task at a time
- **CRITICAL**: Mark completed tasks as [x] in tasks.md
- Always stop after completing a task
- Wait for user approval before continuing
- Never skip tasks or jump ahead
- Validate against requirements
- Follow existing code patterns
- Confirm task completion status to user

## Critical Workflow Rules

### Approval Workflow
- **NEVER** proceed to the next phase without explicit user approval
- Accept only clear affirmative responses: "yes", "approved", "looks good", etc.
- If user provides feedback, make revisions and ask for approval again
- Continue revision cycle until explicit approval is received

### Task Execution
- **ONLY** execute one task at a time during implementation
- **CRITICAL**: Mark completed tasks as [x] in tasks.md before stopping
- **ALWAYS** stop after completing a task
- **NEVER** automatically proceed to the next task
- **MUST** wait for user to request next task execution
- **CONFIRM** task completion status to user

### Task Completion Protocol
When completing any task during `/spec:6_execute`:
1. **Update tasks.md**: Change task status from `- [ ]` to `- [x]`
2. **Confirm to user**: State clearly "Task X has been marked as complete"
3. **Stop execution**: Do not proceed to next task automatically
4. **Wait for instruction**: Let user decide next steps

### Requirement References
- **ALL** tasks must reference specific requirements using _Requirements: X.Y_ format
- **ENSURE** traceability from requirements through design to implementation
- **VALIDATE** implementations against referenced requirements

### Phase Sequence
- **MUST** follow 1_Create → 2_Research → 3_Requirements → 4_Design → 5_Tasks → 6_Execute order
- **Architecture** phase is run once or as needed
- **CANNOT** skip phases or combine phases
- **MUST** complete each phase before proceeding

## File Structure Management

The workflow automatically creates and manages:

```
.documentation/
├── architecture.md           # Global codebase architecture (NEW)
├── features/                 # ← Feature documentation go here
│   └── {feature-name}/
│       ├── context.md        # Feature context from create phase
│       ├── research.md       # Feature-specific research (NEW)
│       ├── requirements.md   # User stories and acceptance criteria
│       ├── design.md        # Technical architecture and design
│       └── tasks.md         # Implementation task breakdown

.claude
├── commands/
│   └── spec/*.md            # Main workflow commands

.spec/
├── templates/
│   └── *-template.md        # Document templates
└── spec-config.json         # Workflow configuration
```

## Error Handling

If issues arise during the workflow:
- **Requirements unclear**: Ask targeted questions to clarify
- **Design too complex**: Suggest breaking into smaller components
- **Tasks too broad**: Break into smaller, more atomic tasks
- **Implementation blocked**: Document the blocker and suggest alternatives

## Success Criteria

A successful spec workflow completion includes:
- ✅ Complete requirements with user stories and acceptance criteria
- ✅ Comprehensive design with architecture and components
- ✅ Detailed task breakdown with requirement references
- ✅ Working implementation validated against requirements
- ✅ All phases explicitly approved by user
- ✅ All tasks completed and integrated

## Getting Started

1. **Initialize**: `/spec:1_create <feature-name> "Description of feature"`
2. **Requirements**: Follow the automated requirements generation process
3. **Design**: Review and approve the technical design
4. **Tasks**: Review and approve the implementation plan
5. **Implementation**: Execute tasks one by one with `/spec:6_execute <task-id>`
6. **Validation**: Ensure each task meets requirements before proceeding

Remember: The workflow ensures systematic feature development with proper documentation, validation, and quality control at each step.
