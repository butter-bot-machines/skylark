# Whitespace Matching Tests

## Headers

# Section    One
Should match "Section One", "SectionOne", "Section  One"

#    Spaced    Header
Should match "Spaced Header", "SpacedHeader", "Spaced   Header"

## Lists

- First    Item
- Second   Item
- Third    Item
  Should match with any spacing

## Paragraphs

This    is    a    paragraph
with    extra    spaces    that
should    match    any    spacing.

This  is  another  paragraph  with
different  spacing  that  should
match  regardless  of  spaces.

## Quotes

> This    quote    has    extra
> spaces    that    should    be
> matched    flexibly

## Tables

| Column    One | Column    Two |
| ------------ | ------------- |
| Value    1   | Value    2    |
| Spaced    Up | Down    Here  |

Should match with any spacing

## Code Blocks

```
function    test()    {
    console.log(   "spaced"   );
    return    true;
}
```

Should match whole block regardless of spaces

## Mixed Content

# Header    With    Spaces
- Item    With    Spaces
> Quote    With    Spaces
```
Code    With    Spaces
```

Should all match with flexible spacing

## Special Cases

# Multiple     Consecutive      Spaces
Should match with any number of spaces

# No Spaces
Should match "NoSpaces" or "No Spaces"

#Tight Spacing#
Should match "Tight Spacing" or "TightSpacing"
