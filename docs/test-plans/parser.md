# Parser Package Test Plan

## Overview

The parser package handles two main responsibilities:
1. Command extraction (lines starting with `!` (exclamation mark))
2. Reference resolution (flexible block matching)

## Components Under Test

### 1. Command Parser
- Pattern: Line starts with `!` (exclamation mark) followed by text, ignoring whitespace
- Assistant Name: First word after exclamation mark `!` (\s*[\w-]+)
    - Case-insensitive matching
    - Normalized to lower-kebab-case
    - Falls back to "default" if unknown
    - Must be trimmed after parsing
- Rest of the line is "prompt text"
- Local Context:
    - Current section's heading and text
    - Parent/sibling headings if relevant
    - Explicitly referenced sections

### 2. Reference Parser
- Block Types:
  - Headers: Markdown headers (# to ######)
  - Lists: Bullet (*/-) and numbered (1.) items
  - Paragraphs: Contiguous text blocks
  - Blockquotes: > prefixed blocks
  - Tables: | delimited rows
  - Code: ```language...``` blocks

- Block Rules:
  - Headers: Include content until next same/higher level
  - Lists: Each item is individual block
  - Paragraphs: Blank line separated
  - Blockquotes: Full quote block
  - Tables: Full table with header
  - Code: Full block with language

- Matching Rules:
  - Partial matches allowed (e.g., "Sec" matches "Section 1")
  - Case insensitive ("section" matches "Section")
  - Whitespace insensitive ("Section1" matches "Section 1")
  - Punctuation insensitive ("Section-1" matches "Section 1")
  - Multiple matches allowed
  - WARN if no matches found

## Test Structure

### 1. Unit Tests (parser_test.go)
```go
// Block type tests
func TestBlockTypes(t *testing.T)
// Matching rule tests
func TestMatchingRules(t *testing.T)
// Warning tests
func TestWarnings(t *testing.T)
```

### 2. Test Data
```
testdata/
├── commands/           # Command parsing tests
│   ├── assistants/    # Assistant name tests
│   │   ├── case.md    # - Case variations
│   │   ├── kebab.md   # - Kebab-case normalization
│   │   └── unknown.md # - Default fallback
│   └── context/       # Context tests
│       ├── current.md # - Current section
│       ├── parent.md  # - Parent headers
│       └── sibling.md # - Sibling sections
└── references/        # Reference parsing tests
    ├── blocks/           # Individual block type tests
    │   ├── headers.md    # Headers with hierarchy
    │   │                 # - Level tracking
    │   │                 # - Content boundaries
    │   │                 # - Nested headers
    │   ├── lists.md      # List variations
    │   │                 # - Bullet/numbered
    │   │                 # - Nested items
    │   │                 # - Mixed formats
    │   ├── paragraphs.md # Paragraph breaks
    │   │                 # - Multi-line
    │   │                 # - With formatting
    │   │                 # - Unicode content
    │   ├── quotes.md     # Blockquote formats
    │   │                 # - Nested quotes
    │   │                 # - With other blocks
    │   │                 # - Indentation
    │   ├── tables.md     # Table structures
    │   │                 # - With alignment
    │   │                 # - Empty cells
    │   │                 # - Escaped pipes
    │   └── code.md       # Code blocks
    │                     # - With language
    │                     # - Nested blocks
    │                     # - Indentation
    ├── matching/         # Matching rule tests
    │   ├── partial.md    # Partial matches
    │   │                 # - Word parts
    │   │                 # - Mixed content
    │   ├── case.md      # Case variations
    │   │                 # - UPPER/lower/Mixed
    │   │                 # - Special cases
    │   ├── space.md     # Whitespace handling
    │   │                 # - Extra spaces
    │   │                 # - No spaces
    │   └── punct.md     # Punctuation handling
    │                     # - Special chars
    │                     # - Mixed formats
    └── mixed/            # Combined tests
        ├── multi.md      # Multiple matches
        │                 # - Cross-block
        │                 # - Overlapping
        └── none.md       # No matches
                         # - Empty refs
                         # - Unicode
                         # - Special chars
```

## Test Cases

### 1. Block Type Tests
```go
tests := []struct {
    name    string
    content string
    query   string
    want    []Block
}{
    // Headers
    {
        name: "header with content",
        content: `# Section 1
Content
## Sub-section
More content
# Section 2`,
        query: "Section 1",
        want: []Block{{
            Type: Header,
            Level: 1,
            Content: "Section 1\nContent\n## Sub-section\nMore content",
        }},
    },
    // Lists
    {
        name: "list items",
        content: `* Item 1
* Item 2
* Item 3`,
        query: "Item",
        want: []Block{
            {Type: List, Content: "Item 1"},
            {Type: List, Content: "Item 2"},
            {Type: List, Content: "Item 3"},
        },
    },
    // Add similar for other block types
}
```

### 2. Matching Rule Tests
```go
tests := []struct {
    name    string
    content string
    query   string
    want    []Block
}{
    // Partial matches
    {
        name: "partial word",
        content: "# Section One",
        query: "Sec",
        want: []Block{{Type: Header, Content: "Section One"}},
    },
    // Case insensitive
    {
        name: "case mismatch",
        content: "# SECTION",
        query: "section",
        want: []Block{{Type: Header, Content: "SECTION"}},
    },
    // Add similar for other rules
}
```

### 3. Warning Tests
```go
tests := []struct {
    name    string
    content string
    query   string
    wantLog string
}{
    {
        name: "no matches",
        content: "# Header\nContent",
        query: "Missing",
        wantLog: "WARN: No blocks matched query 'Missing'",
    },
}
```

## Success Criteria

### 1. Block Types
- ✓ Headers follow hierarchy
- ✓ Lists are individual
- ✓ Paragraphs are complete
- ✓ Blockquotes are complete
- ✓ Tables are complete
- ✓ Code blocks are complete

### 2. Matching Rules
- ✓ Partial matches work
- ✓ Case insensitive
- ✓ Whitespace insensitive
- ✓ Punctuation insensitive
- ✓ Multiple matches work
- ✓ Warnings on no match

## Implementation Strategy

1. Block Parser
   - Implement each block type
   - Handle block boundaries
   - Extract block content

2. Reference Matcher
   - Implement fuzzy matching
   - Handle multiple matches
   - Add warning system

3. Testing
   - Test each block type
   - Test each matching rule
   - Verify warnings

## Notes

1. Block types are well-defined
2. Matching is flexible but predictable
3. Warnings help users understand matches
4. Test data covers all cases
