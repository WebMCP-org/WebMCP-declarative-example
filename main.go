package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "crypto/rand"
    "encoding/hex"
    _ "modernc.org/sqlite"
)

type Todo struct {
    ID          int    `json:"id"`
    Description string `json:"description"`
    Completed   bool   `json:"completed"`
}

var db *sql.DB

const pageHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Todo WebMCP Demo</title>
    <style>
        body { font-family: system-ui; max-width: 600px; margin: 40px auto; padding: 20px; }
        form { margin: 20px 0; }
        input[type="text"] { padding: 8px; width: 300px; }
        button { padding: 8px 16px; }
        .todo { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
        .completed { text-decoration: line-through; opacity: 0.6; }
        .user-info { background: #f5f5f5; padding: 10px; margin-bottom: 20px; font-size: 12px; }
    </style>
    <script src="/polyfill.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/zod@3.22.4/lib/index.umd.js"></script>
    <script src="/webmcp-translator.js"></script>
</head>
<body>
    <div class="user-info">Session: {{.UserID}}</div>
    <h1>Todos</h1>

    <form action="/todos" method="post" tool-name="add-todo" tool-description="Add a new todo item">
        <input type="text" name="description" required placeholder="What needs to be done?"
               tool-param-description="The text of the todo item">
        <button type="submit">Add Todo</button>
    </form>

    <a href="/todos" tool-name="list-todos" tool-description="Get all todo items">Refresh</a>

    <div id="todos">
        {{range .Todos}}
        <div class="todo {{if .Completed}}completed{{end}}">
            <strong>#{{.ID}}</strong>: {{.Description}}

            <form action="/todos/{{.ID}}/toggle" method="post" style="display:inline"
                  tool-name="toggle-todo-{{.ID}}" tool-description="Toggle completion status">
                <button type="submit">{{if .Completed}}Undo{{else}}Complete{{end}}</button>
            </form>

            <form action="/todos/{{.ID}}/delete" method="post" style="display:inline"
                  tool-name="delete-todo-{{.ID}}" tool-description="Delete this todo">
                <button type="submit">Delete</button>
            </form>
        </div>
        {{end}}
    </div>
</body>
</html>
`

const agentResponseHTML = `
<!DOCTYPE html>
<html>
<body>
<script type="application/json" id="agent-response">%s</script>
<meta http-equiv="refresh" content="0;url=/">
</body>
</html>
`

func init() {
    var err error
    db, err = sql.Open("sqlite", "./todos.db")
    if err != nil {
        log.Fatal(err)
    }

    db.Exec(`CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id TEXT NOT NULL,
        description TEXT NOT NULL,
        completed BOOLEAN DEFAULT 0
    )`)
}

func main() {
    http.HandleFunc("/", handleIndex)
    http.HandleFunc("/todos", handleTodos)
    http.HandleFunc("/todos/", handleTodoAction)
    http.HandleFunc("/polyfill.js", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/javascript")
        http.ServeFile(w, r, "polyfill.js")
    })
    http.HandleFunc("/webmcp-translator.js", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/javascript")
        http.ServeFile(w, r, "webmcp-translator.js")
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    fmt.Printf("Server running on http://localhost:%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(w, r)
    todos := getTodos(userID)

    if r.URL.Query().Get("agent") == "true" {
        data, _ := json.Marshal(map[string]interface{}{
            "todos": todos,
            "count": len(todos),
        })
        fmt.Fprintf(w, agentResponseHTML, data)
        return
    }

    tmpl := template.Must(template.New("page").Parse(pageHTML))
    tmpl.Execute(w, map[string]interface{}{
        "UserID": userID[:8],
        "Todos":  todos,
    })
}

func handleTodos(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(w, r)

    switch r.Method {
    case "GET":
        todos := getTodos(userID)
        if r.URL.Query().Get("agent") == "true" {
            data, _ := json.Marshal(map[string]interface{}{
                "todos": todos,
                "count": len(todos),
            })
            fmt.Fprintf(w, agentResponseHTML, data)
        } else {
            http.Redirect(w, r, "/", http.StatusSeeOther)
        }

    case "POST":
        r.ParseForm()
        result, _ := db.Exec("INSERT INTO todos (user_id, description) VALUES (?, ?)",
            userID, r.FormValue("description"))
        id, _ := result.LastInsertId()

        if r.URL.Query().Get("agent") == "true" {
            todo := Todo{int(id), r.FormValue("description"), false}
            data, _ := json.Marshal(map[string]interface{}{
                "created": todo,
                "success": true,
            })
            fmt.Fprintf(w, agentResponseHTML, data)
        } else {
            http.Redirect(w, r, "/", http.StatusSeeOther)
        }
    }
}

func handleTodoAction(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(w, r)
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) < 4 {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }

    id, _ := strconv.Atoi(parts[2])
    action := parts[3]

    var success bool
    switch action {
    case "toggle":
        res, _ := db.Exec("UPDATE todos SET completed = NOT completed WHERE id = ? AND user_id = ?", id, userID)
        rows, _ := res.RowsAffected()
        success = rows > 0
    case "delete":
        res, _ := db.Exec("DELETE FROM todos WHERE id = ? AND user_id = ?", id, userID)
        rows, _ := res.RowsAffected()
        success = rows > 0
    }

    if r.URL.Query().Get("agent") == "true" {
        data, _ := json.Marshal(map[string]interface{}{
            "action": action,
            "id": id,
            "success": success,
        })
        fmt.Fprintf(w, agentResponseHTML, data)
    } else {
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
}

func getTodos(userID string) []Todo {
    rows, _ := db.Query("SELECT id, description, completed FROM todos WHERE user_id = ? ORDER BY id", userID)
    defer rows.Close()

    var todos []Todo
    for rows.Next() {
        var t Todo
        rows.Scan(&t.ID, &t.Description, &t.Completed)
        todos = append(todos, t)
    }
    return todos
}

func getUserID(w http.ResponseWriter, r *http.Request) string {
    cookie, err := r.Cookie("user_id")
    if err != nil {
        bytes := make([]byte, 16)
        rand.Read(bytes)
        userID := hex.EncodeToString(bytes)
        http.SetCookie(w, &http.Cookie{
            Name:     "user_id",
            Value:    userID,
            Path:     "/",
            MaxAge:   86400 * 365,
            HttpOnly: true,
        })
        return userID
    }
    return cookie.Value
}