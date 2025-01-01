# Parent Header Context Tests

## Top Level Section
This is a top level section.

### Child Section
This is a child section.

!assistant analyze this section
Should include parent section content.

### Another Child
More child content.

!assistant analyze this section
Should include parent section content.

## Multiple Levels
Top level content.

### Level Two
Middle level content.

#### Level Three
Deep level content.

!assistant analyze this section
Should include all parent content.

## Parent with Lists
- Parent list item 1
- Parent list item 2

### Child with Quote
> Child quote content
> goes here

!assistant analyze this section
Should include parent list.

## Parent with Code
```
Parent code block
goes here
```

### Child with List
- Child list item 1
- Child list item 2

!assistant analyze this section
Should include parent code.

## Multiple Children
Parent section content.

### First Child
First child content.

!assistant analyze first child
Should include parent content.

### Second Child
Second child content.

!assistant analyze second child
Should include parent content.

## Complex Hierarchy
Top section.

### Middle Section
Middle content.

#### Deep Section One
First deep content.

!assistant analyze this section
Should include all parents.

#### Deep Section Two
Second deep content.

!assistant analyze this section
Should include all parents.
