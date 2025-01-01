# Story: Implement Parser Logic to Match Test Plan (✓ Completed)

## Status

✓ Completed on January 1, 2024 at 12:30

- Implemented command parsing with assistant name handling
- Added block type detection and boundary handling
- Added flexible reference matching system
- Implemented context assembly with parent/sibling tracking
- Added comprehensive test coverage

## Context

The parser package is responsible for:

1. Extracting commands from markdown files
2. Resolving references to content blocks
3. Managing context for commands

Current implementation status:

1. ✓ Basic command pattern defined
2. ✓ Reference pattern defined
3. ✓ Test plan created
4. ✓ Parser logic matches test plan
5. ✓ Tests implemented and passing

## Goal

Update the parser implementation to match the test plan, ensuring:

1. Proper assistant name handling
2. Flexible reference matching
3. Correct block type handling
4. Context management

## Requirements

1. Assistant Name Handling:

   - Case-insensitive matching
   - Simple lowercase normalization
   - Default assistant fallback
   - Whitespace tolerance

2. Block Type Support:

   - Headers with levels
   - Lists (bullet/numbered)
   - Paragraphs
   - Blockquotes
   - Tables
   - Code blocks

3. Reference Matching:

   - Partial matches
   - Case insensitive
   - Whitespace insensitive
   - Punctuation insensitive
   - Multiple matches allowed
   - Warning on no matches

4. Context Rules:
   - Current section content
   - Parent header hierarchy
   - Sibling section content

## Technical Changes

1. Parser Types:

```go
type BlockType int

const (
    Header BlockType = iota
    List
    Paragraph
    Quote
    Table
    Code
)

type Block struct {
    Type    BlockType
    Level   int    // For headers
    Content string
}

type Command struct {
    Assistant  string
    Text       string
    Original   string
    References []string
    Context    map[string]Block
}
```

2. Parser Methods:

```go
type Parser struct {
    commandPattern *regexp.Regexp
    refPattern     *regexp.Regexp
    warnings       []string
}

func (p *Parser) ParseCommand(line string) (*Command, error)
func (p *Parser) ParseBlocks(content string) []Block
func (p *Parser) MatchBlocks(blocks []Block, ref string) []Block
func (p *Parser) AssembleContext(blocks []Block, currentIndex int) []Block
```

## Success Criteria

1. Command Parsing:

```markdown
!ASSISTANT help -> assistant="assistant"
!Test-Name help -> assistant="test-name"
!unknown help -> assistant="default"
```

2. Block Matching:

```markdown
# Section One

Content here.

!analyze # Sec

> Matches "Section One" block

!analyze # ONE

> Matches "Section One" block (case insensitive)

!analyze # missing

> WARN: No blocks matched query 'missing'
```

3. Context Handling:

```markdown
# Parent

## Current

Content here.

!analyze this

> Includes:
>
> - Parent section
> - Current section
> - Any siblings
```

## Testing Plan

1. Unit Tests:

   - Assistant name normalization
   - Block type detection
   - Reference matching
   - Context assembly

2. Test Data:
   - Command variations
   - Block type examples
   - Matching scenarios
   - Context cases

## Acceptance Criteria

1. Command Parsing:

   - ✓ Assistant names normalized correctly
   - ✓ Default fallback works
   - ✓ Whitespace handled properly

2. Block Detection:

   - ✓ All block types recognized
   - ✓ Block boundaries correct
   - ✓ Content preserved

3. Reference Matching:

   - ✓ Partial matches work
   - ✓ Case insensitive
   - ✓ Multiple matches supported
   - ✓ Warnings on no matches

4. Context Handling:
   - ✓ Current section included
   - ✓ Parent headers included
   - ✓ Sibling sections included

## Logging

```
time=2024-01-01T12:30:00Z level=DEBUG msg="parsed command with assistant" assistant=command text=text
time=2024-01-01T12:30:00Z level=DEBUG msg="created command" assistant=command text=text original="!command text"
time=2024-01-01T12:30:00Z level=WARN msg="No blocks matched query 'Missing'"
```
