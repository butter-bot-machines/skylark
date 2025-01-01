# ADR: Story Naming Convention

## Context

We need a consistent way to name story files that provides:

- Unique identifiers
- Chronological ordering
- Type categorization
- Clear description
- Easy parsing

## Decision

Story filenames will follow the pattern:

```
YYYYMMDDHHMM-type-description.md
```

Where:

- YYYYMMDDHHMM: Timestamp in 24-hour format

  - YYYY: Year (2024)
  - MM: Month (01-12)
  - DD: Day (01-31)
  - HH: Hour (00-23)
  - MM: Minute (00-59)

- type: One of the following (strictly enforced):
  - story: New feature or significant change
  - chore: Maintenance, cleanup, or tooling
  - bug: Bug fix
  - docs: Documentation update
  - test: Test improvements
  - refactor: Code restructuring

For example:

```
202401010146-chore-fix-cpu-profile-warning.md
202401010242-chore-fix-tool-naming.md
202401010341-story-implement-assistant-provider-integration.md
```

## Consequences

### Positive

- Natural chronological sorting
- Easy to identify story types
- Consistent format for tooling
- Clear timestamps for tracking

### Negative

- Longer filenames
- Manual timestamp management

## Notes

- Timestamps use 24-hour time
- Types are lowercase
- Descriptions use hyphens
- All files end in .md
