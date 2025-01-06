Below is a comprehensive technical specification for the "Skai" system, pulling together all prior concepts in a clear, structured manner. The goal is to provide a single source of truth for both developers and project collaborators to understand how Skai is built, how it operates, and how to extend it.

*Skai Technical Specification*
*1. Overview*
Skai is a command-line tool (CLI) and file-watching system, written in Go, that augments Markdown-based documents with AI-driven transformations. Users invoke Skai by writing inline "commands" in their documents. These commands reference "assistants" (each defined in a simple folder structure under a hidden directory named `.skai/`) and may optionally specify in-document references for context. Skai detects file changes and automatically processes these commands, inserting AI-generated output back into the user's original files.

*2. Core Concepts*
1. *Assistants*
    * Defined as folders in `.skai/assistants/<assistant_name>`.
    * Each assistant requires at least one file: `prompt.md`, which acts as the "system prompt."
    * Assistants may include a `knowledge` subdirectory for domain-specific additional resources.
    * Assistant names are case-insensitive. Skai normalizes them to lower-kebab-case.
    * If a command references an unknown assistant, Skai falls back to the default assistant.
2. *Commands*
    * A "command" is any line in a Markdown file that begins with an exclamation mark (`!`).
    * If the first word after `!` matches an assistant name (e.g., `researcher`), Skai routes the request to that assistant.
    * If there is no assistant match, Skai uses the default assistant (`.skai/assistants/default`).
    * The text following the assistant name (or following the exclamation mark if no name is present) is treated as the user's prompt.
    * Example:
        * `!researcher Tell me about recent developments in robotics.`
        * `!What's the current date?` (invokes default assistant).
3. *Context References*
    * Commands can include references to other sections in the same file by specifying headers, e.g., `# My Header Title`.
    * Skai searches the Markdown for the header named "My Header Title" (case-insensitive but otherwise exact match for punctuation/spaces).
    * The content of that referenced section is added to the prompt context.
    * Example:
        * `!Analyze findings from # Market Research`
4. *File-Watching*
    * Skai monitors the user's project files whenever they change (on save) and automatically processes commands.
    * Commands are resolved, AI is called, and output is inserted back into the same file.
5. *Tools*
    * Tools are small Go programs stored in `.skai/tools/<tool_name>/`.
    * Each tool has:
        * A single `main.go` file (and optionally more files if needed).
        * A compiled binary matching the folder or tool name (e.g., `currentdatetime`), generated automatically by Skai.
    * Tools communicate with Skai via stdin/stdout in a small JSON or text-based protocol.
    * Example usage:
```
!What time is it?
```
* Skai automatically compiles tools when files in their directory change (e.g., `main.go` is updated).

*3. Folder Structure*
When the user initializes a Skai project (via "skai init <project_name>"), a hidden `.skai` directory is created, populated with some defaults. Example:
```
my_project/
 ├─ .skai/
 │   ├─ assistants/
 │   │   ├─ default/
 │   │   │   ├─ prompt.md
 │   │   │   └─ knowledge/        (optional)
 │   │   └─ researcher/
 │   │       ├─ prompt.md
 │   │       └─ knowledge/        (optional)
 │   ├─ tools/
 │   │   ├─ currentdatetime/     # Built-in tool
 │   │   │   ├─ main.go
 │   │   │   ├─ currentdatetime (compiled binary)
 │   │   └─ url_lookup/         # Custom tool
 │   │       ├─ main.go
 │   │       ├─ url_lookup (compiled)
 │   └─ config.yml
 ├─ outline.md                    (user's own documents)
 ├─ notes.md                      (user's own documents)
 └─ ...
```
 *3.1 The ".skai" Directory*
* "assistants/"
    * Each assistant has its own folder named after the assistant.
    * At minimum, an assistant has `prompt.md`; optionally, a `knowledge` folder or additional files for domain data.
    * Example `prompt.md`:
```
You are a research assistant. Your task is to find relevant information on the web and provide insights in Markdown form.
```
* "tools/"
    * Each tool is in its own folder.
    * Example: `.skai/tools/word_count/main.go` plus the compiled `.skai/tools/word_count/word_count`.
* "config.yml"
    * Reserved for future expansions (e.g., specifying a default model key, concurrency limits, or advanced settings).
*4. Command Parsing & Execution Flow*
When the user saves a Markdown file (e.g., `notes.md`), Skai's workflow is:
1. *Scan for Commands*
    * Skai searches for lines beginning with `!`.
    * It parses the first token after `!`; if it matches an assistant name (case-insensitive) among `.skai/assistants`, that assistant is selected.
    * If no match, the default assistant is used.
    * The rest of the line is the user's prompt.
2. *Resolve References*
    * Skai looks for references of the form `# Header Title` in the prompt text.
    * It collects content from each referenced header. This is appended to the local context for the assistant to consider.
3. *Assemble Context*
    * By default, Skai includes:
        * The current section's heading and text.
        * Potentially parent or sibling headings if relevant (based on local context rules).
        * Any references explicitly included (`# Some Header`).
    * This combined context is fed into the assistant's system prompt.
4. *Assistant Invocation*
    * Skai reads `prompt.md` for the selected assistant to form the system prompt.
    * If the assistant references tools, Skai makes these tools available for the AI's usage (depending on how the AI is invoked; e.g., via function calls if using a bridging approach, or via text-based instructions if directly calling an API).
    * The user's prompt (plus references) is appended.
    * Skai calls the AI model (e.g., OpenAI GPT-4), specifying the system prompt and user text.
5. *AI Response & Insertion*
    * Skai receives the AI's response.
    * Skai inserts that text back into the Markdown file directly under the command that triggered it.
    * Output is typically formatted as:
```
!assistant_name <prompt text>
> <Generated Output>
```
* Or any other specified format that Skai enforces (e.g., collapsible sections or inline comments).
*5. Tools in Detail*
*5.1 Intention & Usage*
* Tools provide specialized functionality for an assistant (e.g., getting current time, fetching URLs, processing data).
* Tools are invoked by the AI model the same way "functions" might be called. With Go, we do not have runtime reflection, so each tool is effectively a CLI command that the user can trigger in their prompt, or that the AI might instruct Skai to run (depending on design).
*5.2 Tool Folder Layout*
Each tool has its own folder under `.skai/tools/`. Example:
```
my_project/.skai/tools/currentdatetime/
 ├─ main.go
 └─ currentdatetime (compiled binary)
```
*5.3 Automatic Compilation*
1. Skai monitors changes to `.go` files in each tool folder.
2. On save, Skai executes something like:
```
go build -o .skai/tools/currentdatetime/currentdatetime .skai/tools/currentdatetime/main.go
```
3. The compiled binary matches the folder name.
4. When a user writes a command like `!What time is it?`, Skai runs `.skai/tools/currentdatetime/currentdatetime` as a subprocess, passing any relevant parameters via stdin.
*5.4 Example "currentdatetime" Tool*
*main.go* (simplified):
```
package main

import (
  "encoding/json"
  "fmt"
  "os"
  "time"
)

type Input struct {
  Format string `json:"format,omitempty"`
}

type Output struct {
  DateTime string `json:"datetime"`
}

func main() {
  data, _ := os.ReadFile(os.Stdin)
  var input Input
  _ = json.Unmarshal(data, &input)

  // Get current time
  now := time.Now()
  format := time.RFC3339
  if input.Format != "" {
    format = input.Format
  }

  // Return formatted time
  output := Output{DateTime: now.Format(format)}
  result, _ := json.Marshal(output)
  fmt.Println(string(result))
}
```
*Usage in Markdown*:
```
!What time is it?
```
*6. Example End-to-End Flow*
1. *Project Initialization*
```
skai init my_project
```
* Creates `.skai/` with default assistant and built-in tools.

2. *User Edits `notes.md`*
```
# Project Notes

*Section 1: Date Check*
!What's today's date?
*Section 2: Robotics Research*
!researcher Tell me about recent developments in humanoid robotics using 5 bullet points.
*Section 3: Analysis*
!Analyze findings from # 2: Robotics Research
```

3. **File Watcher Trigger**  
- On save, Skai detects changes in `notes.md`.
4. **Command Detection**  
- **Section 1**: `!What's today's date?`  
  - No assistant name → default assistant used.  
- **Section 2**: `!researcher Tell me ...`  
  - "researcher" is recognized, so `.skai/assistants/researcher/prompt.md` is used.  
- **Section 3**: `!Analyze findings from # 2: Robotics Research`  
  - No assistant name → default assistant.  
  - Reference to `# 2: Robotics Research` includes Section 2 content as context.

5. **Assistant Invocation & AI Calls**  
- Skai merges the assistant's system prompt (from `prompt.md`) with the user's prompt text, plus any references.  
- The AI processes each command, potentially calling any relevant tools.  
- Results are returned to Skai.

6. **Document Update**  
- Skai inserts each AI response directly under the corresponding command.  
- Example appended text:  
  ```
  !What's today's date?
  > The current date and time is 2024-01-05T10:00:00Z
  ```

## 7. Configuration & Customization

1. **`.skai/config.yml`**  
- Reserved for advanced or global settings (like an API key, concurrency, or default assistant name if "default" is changed).  

2. **Adding New Assistants**  
- Create a folder under `.skai/assistants/<assistant_name>` with at least `prompt.md`.  
- Write system instructions in `prompt.md`.  
- Optionally add a `knowledge/` folder with additional domain data.

3. **Adding or Modifying Tools**  
- Create a new folder under `.skai/tools/<tool_name>`.  
- Write a Go program (e.g., `main.go`).  
- Skai auto-compiles on save, producing a binary named `<tool_name>`.

## 8. Security & Error Handling

1. **Compilation Errors**  
- If a tool's source code fails to compile, Skai should log or print the compiler errors.  
- The old binary remains until a successful build is completed.

2. **Assistant Missing**  
- If a user references an assistant that doesn't exist (e.g., `!Nonexistent fix my text`), Skai logs a warning and uses `default`.

3. **Token or Rate Limits**  
- The system prompt and user content must fit within the model's token limit.  
- Handling of partial context or truncation policy can be defined in `.skai/config.yml`.

4. **Sandboxing Tools** (optional future enhancement)  
- Tools might be restricted from filesystem or network access beyond specific whitelisted endpoints or directories.

## 9. Advantages & Trade-Offs

- **Extraordinarily Simple Workflow**  
- Users write inline commands in Markdown; they are processed automatically on save.  
- No separate "workspace" concept—users own their root, only `.skai/` is reserved.

- **Golang Core**  
- Fast, compiled, self-contained binaries.  
- Tools are easily distributed as single binaries.  
- Some loss of dynamic reflection compared to Python or Node, but the single-file approach plus auto-compilation offsets this.

- **Assistant & Tools Modularity**  
- Each assistant or tool is self-contained.   - Users can easily add domain-specific knowledge under each assistant or write custom tools in Go.

- **Expandable**  
- Could integrate additional features like concurrency (parallel AI calls) or cross-document references.

## 10. Conclusion

Skai is designed to be a robust, low-friction solution for incorporating AI-driven transformations directly into Markdown-based workflows. With a Golang core for performance and portability, an intuitive inline command syntax, and straightforward extension points for assistants and tools, Skai aims to provide both power and simplicity for technical teams.

This specification covers all major components: folder structure, command parsing, tool invocation, assistant definition, and default behaviors for context references. Future enhancements can build on this solid foundation, introducing concurrency, advanced error handling, or deeper domain integration.
