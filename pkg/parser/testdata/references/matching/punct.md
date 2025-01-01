# Punctuation Matching Tests

## Headers

# Section-One

Should match "Section One", "Section.One", "Section_One"

# Complex.Header!With@Punctuation#

Should match "Complex Header With Punctuation"

## Lists

- First-item
- Second.item
- Third_item
- Fourth!item
  Should match with or without punctuation

## Paragraphs

This-is-a-paragraph.with.punctuation
that_should_match_without_punctuation
and!also@match#special$chars.

Another paragraph with (parentheses),
[brackets], {braces}, and <angles>
should match without any punctuation.

## Quotes

> This-is-a-quote.with.punctuation
> that_should_match_flexibly!and@
> ignore#special$characters%here

## Tables

| Header-One | Header.Two |
| ---------- | ---------- |
| Value-1    | Value.2    |
| Item_A     | Item_B     |

Should match with or without punctuation

## Code Blocks

```
function-name() {
    console.log("test.message");
    return_value();
}
```

Should match whole block ignoring punctuation

## Mixed Content

# Header-With.Punctuation

- Item_With_Underscores
  > Quote!With@Special#Chars

```
Code-With.Mixed_Punctuation
```

Should all match flexibly

## Special Cases

# Multiple!!!Consecutive!!!Punctuation

Should match with any punctuation removed

# No.Punctuation.At.All

Should match "No Punctuation At All"

# Mixed-Case.With_Punctuation!

Should match case and punctuation variations
