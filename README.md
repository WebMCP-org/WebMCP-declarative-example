# Declarative API for WebMCP - Go Demo

Credit to [EisenbergEffect](https://github.com/EisenbergEffect) for the [original idea](https://github.com/webmachinelearning/webmcp/issues/22#issuecomment-3263168311)

This repository explores the feasibility of a declarative HTML-based approach for WebMCP tools, as an alternative/complement to the JavaScript API.

## Background

This demo was created in response to discussions about whether WebMCP tools could be defined declaratively through HTML attributes rather than JavaScript registration. The idea: existing HTML forms and links could become MCP tools by simply adding attributes.

## The Approach

Instead of JavaScript tool registration:
```javascript
window.mcp.registerTool('add-todo', {...}, async (params) => {...})
```

We explored declarative HTML:
```html
<form action="/todos" method="post" tool-name="add-todo" tool-description="Add a new todo item">
  <input type="text" name="description" required tool-param-description="The text of the todo item">
  <button type="submit">Add Todo</button>
</form>
```

## Demo Implementation

The demo consists of:
- **Go server** (`main.go`) - A simple todo app with SQLite storage
- **JavaScript translator** (`webmcp-translator.js`) - Automatically discovers HTML elements with `tool-*` attributes and registers them as MCP tools
- **Multi-tenant support** - Each browser session gets its own todo list via cookies

## Key Findings

### Challenges

1. **Tool explosion** - Having individual tools for each list item (e.g., delete-todo-1, delete-todo-2) quickly pollutes the agent's context. We had to consolidate to just 4 generic tools.

2. **Response control** - The declarative approach struggles to maintain functional equivalence between UI interactions and tool executions. We needed workarounds (hidden iframes, query parameters) to handle both HTML responses for users and JSON for agents.

3. **Schema limitations** - HTML forms can't express rich schemas (arrays, nested objects, complex validation) that tools often require.

4. **No advanced patterns** - Can't handle streaming responses, long-running operations, or stateful tools.

### Strengths

- **Simple read operations** - Links as tools (`<a href="/todos" tool-name="list-todos">`) are genuinely simple
- **Progressive enhancement** - Forms work normally even without WebMCP support
- **Lower barrier to entry** - No JavaScript knowledge required for basic tools

## Running the Demo

```bash
go mod init todo-demo
go get modernc.org/sqlite
go run main.go
```

Visit http://localhost:8080

## Conclusion

While the declarative approach has appeal for simple cases, this exploration demonstrates why WebMCP chose a JavaScript-first approach. The declarative model looks clean in simple demos but breaks down when real-world functionality is needed.

The declarative approach might work as a complementary layer for simple CRUD operations, but can't serve as the foundation for a flexible tool system.

## Related Discussion

This implementation was created as part of the conversation [here](https://github.com/webmachinelearning/webmcp/issues/22) exploring declarative alternatives for WebMCP.