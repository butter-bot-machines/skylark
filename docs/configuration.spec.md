# Skylark Configuration Specification

## Assistants
1. Definition:
    * Assistants are defined by a prompt.md file in their respective folder within .skai/assistants/.
    * The assistant name is inferred from the folder name (case-insensitive, normalized to lower-kebab-case).
    * Each assistant consists of:
        * Metadata in front-matter.
        * Prompt content for behavioral instructions.
2. Front-Matter Specification:
```yaml

description: <assistant_description>
model:
  name: [<provider_name>:]<model_name> # Optional provider.
  temperature: 0.7
  max_length: 4095
  #... other model parameter overries
tools:
  - name: <name-lower-kebab-case>
    description: <tool_description> # Optional, assistant-specific tool description.

after the last  is all the prompt content
```
3. Details:
    * Assistant Name: Inferred from the folder structure.
    * Model Configuration:
        * Combines provider and model into a single field (name).
        * Allows for nested model parameters like temperature and max_length.
    * Tool Overrides:
        * Tools are specified as a list of objects, each containing the tool's name and an optional description field to override its default description.
4. Prompt Content:
    * The Markdown body following the front-matter provides system instructions for assistant behavior.
5. Example Assistant File:
```yaml

description: A research-focused assistant that provides concise summaries and insights.
model:
  name: openai:gpt-4
  temperature: 0.3
  max_length: 2000
tools:
  - name: web_search
    description: Performs targeted web searches for academic materials.
  - name: summarize
    description: Condenses technical reports into short, digestible summaries.

You are an expert research assistant specializing in synthesizing complex information into clear, concise explanations. Always respond formally and include sources when applicable. Format your responses with Markdown headings and bullet points for clarity.
```
## Tools
1. Definition:
    * Tools are small programs written in Go, stored in .skai/tools/<tool_name>/, and compiled by Skai automatically.
    * They must adhere to a strict interface and implement two commands:
        * --usage: Outputs tool schema and runtime requirements.
        * --health: Verifies operational readiness.
2. Tool Schema Specification (--usage Output):
    * Tools output a JSON descriptor with two fields:
        1. schema: OpenAI-compatible function definition including the tool’s name, description, and input parameters.
        2. env: Key-value pairs defining required runtime environment variables, each with:
            * type: Data type of the variable.
            * description: Explanation of its purpose.
            * default: Optional default value.
3. Tool Health Check (--health Output):
    * Returns a boolean status or a JSON object indicating readiness.
    * Example Output:
```json
{
  "status": true,
  "details": "All required environment variables are set."
}
```
4. Example --usage Output:
```json
{
  "schema": {
    "name": "web_search",
    "description": "Conduct broad-based web inquiries.",
    "parameters": {
      "type": "object",
      "properties": {
        "queries": {
          "type": "array",
          "items": { "type": "string" },
          "description": "Varied topics requiring diverse viewpoints or deeper exploration."
        }
      },
      "required": ["queries"],
      "additionalProperties": false
    }
  },
  "env": {
    "API_KEY": {
      "type": "string",
      "description": "Authentication key for the web search API.",
      "default": null
    },
    "TIMEOUT": {
      "type": "integer",
      "description": "Request timeout in seconds.",
      "default": 30
    }
  }
}
```
5. Conventions:
    * Tool names must follow lower-kebab-case.
    * Tools reside in .skai/tools/ and are maintained by users.
6. Compilation:
    * Skai automatically compiles tools when their source files (main.go) are modified.
    * Compiled binaries match the folder name (e.g., .skai/tools/summarize/summarize).

## Config
1. Definition:
    * A centralized configuration file (config.yaml) stores runtime values for models and tools.
2. Structure:
```yaml
models:
  <provider_name>:
    <model_name>:
      api_key: <api_key>
      temperature: <default_temperature>
      max_tokens: <default_max_tokens>
tools:
  <tool_name>:
    env:
      <name>: <value>
```
3. Details:
    * Models and tools reference their configurations in this file.
    * Environment variables (env) for tools are explicitly defined here.
4. Example Config File:
```yaml
models:
  openai:
    gpt-4:
      api_key: sk-XXXXXX
      temperature: 0.7
      max_tokens: 1500
    gpt-3.5-turbo:
      api_key: sk-YYYYYY
      temperature: 0.5
      max_tokens: 1000
tools:
  web_search:
    env:
      API_KEY: websearch-KEY
      TIMEOUT: 30
  summarize:
    env:
      API_KEY: summarize-KEY
```

**Execution Flow**
1. Skai interrogates tools via --usage to determine runtime dependencies (schema and env).
2. Skai verifies tool health using the --health command before execution.
3. Skai retrieves configured environment variables from config.yaml.
4. If required variables are missing, Skai attempts to resolve them dynamically using defaults specified in the tool’s --usage. Warnings are issued for unresolved variables.
5. Skai invokes tools with their resolved environments and passes results back to the invoking assistant or workflow.
