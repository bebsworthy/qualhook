# Spec Create Command

Create a new feature specification following the spec-driven workflow.

## Usage
```
/spec:1_create <feature-name> [description]
```

## Instructions
You are helping create a new feature specification. Follow these steps:

**WORKFLOW SEQUENCE**: 1_Create → 2_Research → 3_Requirements → 4_Design → 5_Tasks → 6_Execute
**NEW**: Research phase now follows create phase for better context.

**FILE LOCATION**: Feature files will be created in `.documentation/features/{feature-name}/`

1. **Check Prerequisites**
   - If `.documentation/architecture.md` doesn't exist:
     - Inform user: "No architecture documentation found. Consider running `/spec:architecture` first for better context."
     - Continue anyway (architecture is helpful but not required)

2. **Create Directory Structure**
   - Create `./documentation/features/{feature-name}` directory
   - Create `context.md` with feature name and description
   - Initialize empty research.md, requirements.md, design.md, and tasks.md files

3. **Save Feature Context**
   - Save feature name and description to `context.md`
   - This will be used by the research phase

4. **Guide to Research Phase**
   - Inform user: "Feature spec created. Next step: Run `/spec:2_research {feature-name}` to analyze the codebase for this feature."
   - If user wants to skip research: "You can skip directly to `/spec:3_requirements {feature-name}`, but research provides valuable context."

5. **Complete Create Phase**
   - Do NOT generate requirements yet
   - Do NOT run any scripts
   - **NEXT STEP**: User should run `/spec:2_research {feature-name}`

6. **Rules**
   - Only create ONE spec at a time
   - Always use kebab-case for feature names
   - Follow the exact EARS format for acceptance criteria
   - Do not proceed without explicit user approval
   - **DO NOT** run scripts during /spec:1_create - only create requirements

## Example
```
/spec:1_create user-authentication "Allow users to sign up and log in securely"
```

## Next Steps
After creating the feature spec:
1. Run `/spec:2_research {feature-name}` to analyze codebase context
2. Or skip to `/spec:3_requirements {feature-name}` (not recommended)
