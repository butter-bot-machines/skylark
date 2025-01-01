# Case Matching Tests

## Headers

# UPPERCASE SECTION

Should match "uppercase", "UPPERCASE", "UpPerCaSe"

# MixedCase Section

Should match "mixedcase", "MIXEDCASE", "MiXeDcAsE"

## Lists

- FIRST ITEM
- Second Item
- THiRd ItEm
  Should all match any case variation

## Paragraphs

THIS IS AN UPPERCASE PARAGRAPH
THAT SHOULD MATCH ANY CASE
LIKE "uppercase" or "UPPERCASE".

This is a MixedCase paragraph
that Should Match any CASE
like "mixedcase" or "MIXEDCASE".

## Quotes

> THIS IS AN UPPERCASE QUOTE
> That Should Match ANY case
> VARIATIONS like "quote" or "QUOTE"

## Tables

| HEADER | Value |
| ------ | ----- |
| UPPER  | CASE  |
| Mixed  | Case  |
| lower  | case  |

Should match any case variation

## Code Blocks

```
FUNCTION UPPERCASE() {
    CONSOLE.LOG("TEST");
}

function MixedCase() {
    console.log("test");
}
```

Should match whole block with any case

## Special Cases

# SQL Query Section

```sql
SELECT * FROM table
WHERE column = 'value'
```

Should match "select", "SELECT", "Select"

## Mixed Content

# SECTION ONE

- ITEM ONE
  > QUOTE ONE

```
CODE ONE
```

Should all match regardless of case
