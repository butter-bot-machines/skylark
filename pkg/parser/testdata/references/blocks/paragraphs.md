# Paragraph Tests

Simple paragraph on
multiple lines should
be treated as one block.

Another paragraph with
multiple lines and some
formatting like *italic*
and **bold** text.

Paragraph with special chars:
@#$% and numbers 123 and
punctuation marks !@#.

This paragraph has a hyphenated-word
and an_underscore_word that should
be matched regardless.

A paragraph followed by a list:
* Should not include list
* In its content

> A paragraph followed by a quote
> Should not include the quote
> In its content

```
A paragraph followed by code
Should not include the code
In its content
```

| A paragraph followed by table |
| Should not include the table |
| In its content |

Paragraph with Unicode: 
こんにちは and 你好 and
café and über should all
be handled properly.

   Paragraph with leading
   spaces should still be
   matched properly.

Paragraph with trailing   
spaces should still be    
matched properly.
