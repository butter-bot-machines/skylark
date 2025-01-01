# Partial Matching Tests

## Headers
# Introduction Section
Should match "Intro", "Section", "duction"

# Implementation Details
Should match "Implement", "Details", "tion Det"

## Lists
* First implementation step
* Second implementation step
* Third implementation step
Should all match "implementation" or "step"

## Paragraphs
This is an introduction paragraph
that should match partial words like
"intro" or "paragraph" or "this is".

Here's another paragraph about
implementation details that should
match parts like "implement" or "details".

## Quotes
> This quote discusses implementation
> and should match parts like "discuss"
> or "implement" or "this quote".

## Tables
| Section | Details |
|---------|---------|
| Intro | First part |
| Implementation | Second part |
Should match "Intro", "First", "part", etc.

## Code Blocks
```
function implementation() {
    console.log("test");
}
```
Should match whole block with "function" or "implementation"

## Mixed Content
# Section About Implementation
* First implementation detail
> Quote about implementation
```
Code with implementation
```
Should all match "implementation"
