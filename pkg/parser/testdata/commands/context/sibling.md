# Sibling Header Context Tests

## First Section
This is the first section
with some content.

## Second Section
This is the second section
that follows the first.

!assistant analyze with siblings
Should include adjacent sections.

## Lists and Quotes
- List item one
- List item two

## Code and Tables
```
Code block here
with content
```

!assistant analyze with siblings
Should include list and code sections.

## Multiple Siblings
First sibling content.

## Current Section
Current section content.

## Next Sibling
Next sibling content.

!assistant analyze with siblings
Should include both siblings.

## Complex Content
### Subsection One
Content in first subsection.

## Adjacent Section
### Subsection A
Content in A.

### Subsection B
Content in B.

!assistant analyze with siblings
Should include complex section.

## Edge Cases

## Empty Sibling

## Another Empty

!assistant analyze with siblings
Should handle empty siblings.

# New Top Level
## First Child
Content in first.

## Second Child
Content in second.

## Third Child
Content in third.

!assistant analyze with siblings
Should include sibling children.

# Mixed Levels
## Level Two A
### Level Three A
Content A.

## Level Two B
### Level Three B
Content B.

!assistant analyze with siblings
Should handle mixed levels.
