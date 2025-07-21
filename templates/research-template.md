# Research: {feature-name}

## 1. Feature Context

### Description
[Brief description of what this feature will do]

### Scope
- **Layer**: [Frontend/Backend/Full-stack]
- **Components**: [High-level components affected]

### Architecture Reference
See [architecture.md](../../../architecture.md) for overall system architecture.

## 2. Similar Existing Features

### Feature: [Name]
- **Location**: `path/to/feature`
- **What it does**: [Brief description]
- **What we can reuse**:
  - [Specific component/pattern]
  - [Specific utility/helper]
- **What to improve**:
  - [Known issue to avoid]
  - [Better approach to use]

### Feature: [Name]
- **Location**: `path/to/feature`
- **Patterns to follow**: [Specific patterns]
- **Lessons learned**: [What worked/didn't work]

## 3. Affected Components

### Direct Impact
| Component | Location | Purpose | Changes Needed |
|:----------|:---------|:--------|:---------------|
| [Name] | `path/to/file` | [What it does] | [What changes] |

### Integration Points
| System | Type | Current State | Integration Needs |
|:-------|:-----|:--------------|:------------------|
| [API/Service] | [REST/Event/etc] | [How it works now] | [What's needed] |

## 4. Technical Considerations

### New Dependencies
- [ ] No new dependencies needed
- [ ] New dependencies required:
  - **Package**: [name] - [why needed]
  - **Version**: [recommended version]

### Performance Impact
- **Expected Load**: [Estimate]
- **Performance Concerns**: [If any]
- **Optimization Needs**: [If any]

### Security Considerations
- **Authentication**: [Required/changes needed]
- **Authorization**: [Permission requirements]
- **Data Sensitivity**: [PII/sensitive data handling]
- **Vulnerabilities**: [Potential security risks]

## 5. Implementation Constraints

### Must Follow (from architecture.md)
- [Specific pattern/convention from architecture]
- [Required integration approach]
- [Mandated tools/libraries]

### Cannot Change
- [Existing code that must remain stable]
- [APIs that can't break]
- [Data structures that are fixed]

### Technical Debt in Area
- [Existing debt that affects this feature]
- [Workarounds that may be needed]
- [Future refactoring to consider]

## 6. Recommendations

### Architecture Approach
Based on the codebase analysis:
- **Pattern**: Use [specific pattern] as seen in [reference]
- **Structure**: Follow [specific structure] from [example]
- **Integration**: Connect via [method] like [example]

### Implementation Strategy
1. **Start with**: [Specific component/file]
2. **Build on**: [Existing code to extend]
3. **Reuse**: [Specific utilities/helpers]
4. **Avoid**: [Known pitfalls]

### Testing Approach
- **Unit Tests**: Follow pattern in `tests/[example]`
- **Integration Tests**: Similar to `tests/[example]`
- **Test Data**: Use existing factories in `[location]`

## 7. Risks and Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|:-----|:------------|:-------|:-----------|
| [Description] | High/Med/Low | High/Med/Low | [How to handle] |

### Integration Risks
- **Breaking Changes**: [What could break]
- **Data Migration**: [If needed]
- **Rollback Plan**: [How to rollback if needed]

## 8. Next Steps

### Requirements Considerations
Based on this research, the requirements should:
- Include [specific requirement]
- Consider [specific constraint]
- Account for [specific integration]

### Design Considerations
The design phase should focus on:
- [Specific design challenge]
- [Integration approach]
- [Performance optimization]

---

## Research Summary

**Key Findings**:
1. [Most important discovery]
2. [Second key finding]
3. [Third key finding]

**Recommended Approach**: [One-sentence summary]

**Proceed to Requirements?** [Yes/No]