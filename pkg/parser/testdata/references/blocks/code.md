# Code Block Tests

## Simple Code Block
```
Simple code block
without language
specification
```

## Language-Specific Blocks
```go
func main() {
    fmt.Println("Hello, Go!")
}
```

```python
def main():
    print("Hello, Python!")
```

```javascript
function main() {
    console.log("Hello, JavaScript!");
}
```

## Code with Special Content
```
Special chars: @#$%
Numbers: 123
Hyphenated-words
Underscore_words
```

## Code with Unicode
```
Japanese: こんにちは
Chinese: 你好
German: über
French: café
```

## Code with Markdown
```
# This is not a header
* This is not a list
> This is not a quote
| This | Is | Not | A | Table |
```

## Code with Indentation
```python
class Example:
    def method(self):
        if True:
            print("Indented")
            return True
```

## Code with Empty Lines
```
First line

Middle line

Last line
```

## Code with Spaces
```
    Leading spaces
Text with    multiple    spaces
    Trailing spaces    
```

## Nested Code Blocks
~~~
Outer block
```
Inner block
```
Outer block
~~~
