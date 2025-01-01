# Current Section Context Tests

## Simple Section
This is a simple section with
multiple lines of content.

!assistant analyze this section
Should include section content

## Section with List
- First point
- Second point
- Third point

!assistant analyze this section
Should include list content

## Section with Quote
> This is a quote
> with multiple lines
> of content

!assistant analyze this section
Should include quote content

## Section with Code
```
function test() {
    console.log("hello");
}
```

!assistant analyze this section
Should include code content

## Mixed Content Section
This section has multiple types:

- A list item
- Another item

> A quote block
> with content

```
Some code
block here
```

!assistant analyze this section
Should include all content types

## Empty Section

!assistant analyze this section
Should handle empty sections

## Command at Start
!assistant analyze this section
This section has the command
at the start instead of end.

## Multiple Commands
This section has multiple
lines of content.

!assistant first command
Should use content above.

!assistant second command
Should use all content including
previous command and response.

## Nested Headers
### Subsection One
Content in subsection.

!assistant analyze this section
Should include subsection content.

### Subsection Two
More content here.

!assistant analyze this section
Should include this subsection.
