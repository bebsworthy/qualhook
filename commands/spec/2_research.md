# Feature Research Command

Perform lightweight, feature-specific research leveraging the global architecture documentation.

## Usage
```
/spec:2_research <feature-name>
```

## Prerequisites
- Must have `.documentation/architecture.md` (run `/spec:architecture` first)
- Must have created the feature spec with `/spec:1_create`

## Instructions

You are performing focused research for a specific feature, building on the existing architecture documentation to avoid redundant analysis.

### Process

1. **Validate Prerequisites**
   - Check if `.documentation/architecture.md` exists
     - If not: "Architecture documentation not found. Please run `/spec:architecture` first."
   - Check if `.documentation/specs/{feature-name}/context.md` exists
     - If not: "Feature spec not found. Please run `/spec:1_create {feature-name}` first."

2. **Load Context**
   - Read `.documentation/architecture.md` for codebase overview
   - Read `.documentation/specs/{feature-name}/context.md` for feature description
   - Understand the feature scope and requirements

3. **Feature-Specific Analysis** (Focused Search Only)
   
   a. **Find Similar Features**
      - Search for existing features with similar functionality
      - Look for patterns that can be reused
      - Identify code that solves similar problems
      - Note what worked well and what to avoid
   
   b. **Identify Affected Components**
      - Based on architecture.md, identify which components will be touched
      - Map out specific files and modules involved
      - Find integration points with existing code
      - Note which APIs or services will be affected
   
   c. **Assess Integration Requirements**
      - Identify specific API endpoints to modify or create
      - Find data models that need updating
      - Locate services that need integration
      - Map authentication/authorization touchpoints
   
   d. **Feature-Specific Constraints**
      - Check for patterns this feature MUST follow (from architecture.md)
      - Identify code that CANNOT be changed
      - Find performance requirements specific to this area
      - Note security considerations for this feature

4. **Technical Considerations**
   - Identify if new dependencies are needed (only for this feature)
   - Find technical debt in the specific area
   - Assess security implications of the feature
   - Consider performance impacts

5. **Generate Research Document**
   - Use the research template from `.spec/templates/research-template.md`
   - Focus on feature-specific findings only
   - Reference architecture.md instead of re-documenting general patterns
   - Keep the document concise (target: under 2000 tokens)

6. **Save and Continue**
   - Save to `.documentation/specs/{feature-name}/research.md`
   - Ask: "Research complete. Proceed to requirements phase? (y/n)"
   - If yes, guide user to run `/spec:3_requirements {feature-name}`

### Key Principles

- **Leverage Existing Knowledge**: Reference architecture.md, don't repeat it
- **Stay Focused**: Only research what's specific to this feature
- **Be Practical**: Find actionable insights, not theoretical analysis
- **Save Tokens**: Keep analysis lightweight and targeted
- **Think Integration**: Focus on how this feature fits into existing code

### What NOT to Include

- General architecture patterns (already in architecture.md)
- Technology stack details (already in architecture.md)
- Overall project structure (already in architecture.md)
- Generic coding conventions (already in architecture.md)

### What TO Include

- Specific similar features and their implementation
- Exact files and components affected
- Feature-specific patterns to follow
- Integration points unique to this feature
- Risks and constraints for this specific feature

### Example Output Focus

Instead of: "The project uses Express.js with middleware pattern"
Write: "The user-profile feature at `/api/users/profile` uses similar validation middleware that we can extend"

Instead of: "The project follows MVC architecture"
Write: "Create the new auth controller in `src/controllers/` following the pattern in `UserController.js`"

## Next Steps

After research approval, proceed to `/spec:3_requirements {feature-name}` to generate requirements informed by the research findings.