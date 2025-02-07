// run main.go server -port=8080
// ./companion-proxy server -port=8080

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Action represents a configured action to forward to Companion
type Action struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// LogEntry tracks the execution results of actions
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	ActionID  string    `json:"action_id"`
	Success   bool      `json:"success"`
	Response  string    `json:"response"`
}

var (
	actions   = make(map[string]Action)
	actionsMu sync.Mutex
	logs      []LogEntry
	logsMu    sync.Mutex
)

// Persistent storage
const dataFile = "companion_proxy_data.json"

var dataMu sync.Mutex

// PersistentData represents the data structure saved to disk
type PersistentData struct {
	Actions map[string]Action `json:"actions"`
	Logs    []LogEntry        `json:"logs"`
}

// MAIN: CLI and Server Dispatcher
func main() {
	// If no command-line arguments are provided, print help.
	if len(os.Args) < 2 {
		printHelp()
		runServer(8080)
		return
	}

	// The FIRST argument determines the subcommand.
	cmd := os.Args[1]
	switch cmd {
	case "help":
		printHelp()
		return
	case "server":
		// Start the HTTP server.
		serverFlags := flag.NewFlagSet("server", flag.ExitOnError)
		port := serverFlags.Int("port", 8080, "Port number for the server")
		serverFlags.Parse(os.Args[2:])
		runServer(*port)
		return
	case "add":
		// Add a new action.
		addFlags := flag.NewFlagSet("add", flag.ExitOnError)
		name := addFlags.String("name", "", "Unique name for the action (required)")
		url := addFlags.String("url", "", "URL for the action (required)")
		method := addFlags.String("method", "POST", "HTTP method to use")
		header := addFlags.String("header", "", "Header in \"Key:Value\" format (optional)")
		body := addFlags.String("body", "", "Request body (optional)")
		addFlags.Parse(os.Args[2:])

		if *name == "" || *url == "" {
			fmt.Println("Error: -name and -url are required.")
			return
		}

		loadFromFile()

		// Check for duplicate name!!
		actionsMu.Lock()
		for _, a := range actions {
			if a.Name == *name {
				actionsMu.Unlock()
				fmt.Println("Error: Action name must be unique.")
				return
			}
		}
		actionsMu.Unlock()

		headers := make(map[string]string)
		if *header != "" {
			parts := strings.SplitN(*header, ":", 2)
			if len(parts) != 2 {
				fmt.Println("Error: Invalid header format. Expected Key:Value")
				return
			}
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}

		newAction := Action{
			ID:      generateID(),
			Name:    *name,
			URL:     *url,
			Method:  *method,
			Headers: headers,
			Body:    *body,
		}

		actionsMu.Lock()
		actions[newAction.ID] = newAction
		actionsMu.Unlock()
		saveToFile()

		fmt.Printf("Added action:\nID: %s\nName: %s\nURL: %s\nMethod: %s\nHeaders: %+v\nBody: %s\n",
			newAction.ID, newAction.Name, newAction.URL, newAction.Method, newAction.Headers, newAction.Body)
		return
	case "edit":
		// Edit an existing action.
		editFlags := flag.NewFlagSet("edit", flag.ExitOnError)
		id := editFlags.String("id", "", "ID of the action to edit (required)")
		name := editFlags.String("name", "", "New name for the action (optional)")
		url := editFlags.String("url", "", "New URL for the action (optional)")
		method := editFlags.String("method", "", "New HTTP method (optional)")
		header := editFlags.String("header", "", "Header in \"Key:Value\" format (optional)")
		body := editFlags.String("body", "", "New request body (optional)")
		editFlags.Parse(os.Args[2:])

		if *id == "" {
			fmt.Println("Error: -id is required for editing an action.")
			return
		}

		loadFromFile()
		actionsMu.Lock()
		action, exists := actions[*id]
		if !exists {
			actionsMu.Unlock()
			fmt.Println("Error: Action not found.")
			return
		}
		// If a new name is provided, check uniqueness.
		if *name != "" && *name != action.Name {
			for _, a := range actions {
				if a.Name == *name {
					actionsMu.Unlock()
					fmt.Println("Error: Action name must be unique.")
					return
				}
			}
			action.Name = *name
		}
		if *url != "" {
			action.URL = *url
		}
		if *method != "" {
			action.Method = *method
		}
		if *header != "" {
			parts := strings.SplitN(*header, ":", 2)
			if len(parts) != 2 {
				actionsMu.Unlock()
				fmt.Println("Error: Invalid header format. Expected Key:Value")
				return
			}
			// Replace the headers with the provided one.
			action.Headers = map[string]string{
				strings.TrimSpace(parts[0]): strings.TrimSpace(parts[1]),
			}
		}
		if *body != "" {
			action.Body = *body
		}
		actions[*id] = action
		actionsMu.Unlock()
		saveToFile()

		fmt.Printf("Edited action:\nID: %s\nName: %s\nURL: %s\nMethod: %s\nHeaders: %+v\nBody: %s\n",
			action.ID, action.Name, action.URL, action.Method, action.Headers, action.Body)
		return
	case "delete":
		// Delete an action.
		deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
		id := deleteFlags.String("id", "", "ID of the action to delete (required)")
		deleteFlags.Parse(os.Args[2:])

		if *id == "" {
			fmt.Println("Error: -id is required for deleting an action.")
			return
		}

		loadFromFile()
		actionsMu.Lock()
		if _, exists := actions[*id]; !exists {
			actionsMu.Unlock()
			fmt.Println("Error: Action not found.")
			return
		}
		delete(actions, *id)
		actionsMu.Unlock()
		saveToFile()

		fmt.Println("Deleted action with id:", *id)
		return
	case "list":
		// List all actions.
		loadFromFile()
		actionsMu.Lock()
		if len(actions) == 0 {
			fmt.Println("No actions found.")
		} else {
			fmt.Println("Actions:")
			for _, a := range actions {
				fmt.Printf("ID: %s\nName: %s\nURL: %s\nMethod: %s\nHeaders: %+v\nBody: %s\n\n",
					a.ID, a.Name, a.URL, a.Method, a.Headers, a.Body)
			}
		}
		actionsMu.Unlock()
		return
	case "trigger":
		// Trigger an action.
		triggerFlags := flag.NewFlagSet("trigger", flag.ExitOnError)
		id := triggerFlags.String("id", "", "ID of the action to trigger")
		name := triggerFlags.String("name", "", "Name of the action to trigger")
		triggerFlags.Parse(os.Args[2:])

		if *id == "" && *name == "" {
			fmt.Println("Error: either -id or -name is required for triggering an action.")
			return
		}

		loadFromFile()
		var action Action
		if *id != "" {
			actionsMu.Lock()
			a, exists := actions[*id]
			actionsMu.Unlock()
			if !exists {
				fmt.Println("Error: Action not found.")
				return
			}
			action = a
		} else if *name != "" {
			actionsMu.Lock()
			for _, a := range actions {
				if a.Name == *name {
					action = a
					break
				}
			}
			actionsMu.Unlock()
			if action.ID == "" {
				fmt.Println("Error: Action not found.")
				return
			}
		}
		// Execute the action.
		success, response, err := executeAction(action)
		if err != nil {
			fmt.Printf("Error triggering action: %v\n", err)
		} else {
			fmt.Printf("Action triggered. Success: %v, Response: %s\n", success, response)
		}
		return
	case "logs":
		// Display logs.
		loadFromFile()
		logsMu.Lock()
		if len(logs) == 0 {
			fmt.Println("No logs found.")
		} else {
			fmt.Println("Logs:")
			for _, l := range logs {
				fmt.Printf("Timestamp: %s, Action ID: %s, Success: %v, Response: %s\n",
					l.Timestamp.Format(time.RFC3339), l.ActionID, l.Success, l.Response)
			}
		}
		logsMu.Unlock()
		return
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printHelp()
		return
	}
}

// Server Setup and HTTP Handlers
func runServer(port int) {
	loadFromFile()

	// Frontend: Serve static files.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	// Define API routes.
	http.HandleFunc("/actions/", handleSingleAction)
	http.HandleFunc("/actions", handleActionsCollection)
	http.HandleFunc("/trigger/", handleTrigger)
	http.HandleFunc("/logs", handleLogs)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server running on port %d\n", port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleActionsCollection handles CRUD operations for actions.
func handleActionsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getActions(w, r)
	case http.MethodPost:
		createAction(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSingleAction supports GET (to retrieve an action), PUT (to update), and DELETE.
func handleSingleAction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getAction(w, r)
	case http.MethodPut:
		updateAction(w, r)
	case http.MethodDelete:
		deleteAction(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTrigger handles triggering actions via GET requests.
func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract action name from URL path.
	actionName := strings.TrimPrefix(r.URL.Path, "/trigger/")
	if actionName == "" {
		http.Error(w, "Action name is required", http.StatusBadRequest)
		return
	}

	// Find action by name.
	var action Action
	actionsMu.Lock()
	for _, a := range actions {
		if a.Name == actionName {
			action = a
			break
		}
	}
	actionsMu.Unlock()

	if action.ID == "" {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	triggerAction(w, r, action.ID)
}

// handleLogs handles fetching logs.
func handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	getLogs(w, r)
}

// HTTP Handlers (for API)
func getActions(w http.ResponseWriter, _ *http.Request) {
	actionsMu.Lock()
	defer actionsMu.Unlock()

	var actionList []Action
	for _, action := range actions {
		actionList = append(actionList, action)
	}

	respondJSON(w, actionList)
}

func getAction(w http.ResponseWriter, r *http.Request) {
	// Extract the action ID from the URL path.
	actionID := strings.TrimPrefix(r.URL.Path, "/actions/")
	if actionID == "" {
		http.Error(w, "Action ID is required", http.StatusBadRequest)
		return
	}

	actionsMu.Lock()
	action, exists := actions[actionID]
	actionsMu.Unlock()

	if !exists {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	respondJSON(w, action)
}

func createAction(w http.ResponseWriter, r *http.Request) {
	var action Action
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actionsMu.Lock()
	for _, a := range actions {
		if a.Name == action.Name {
			actionsMu.Unlock()
			http.Error(w, "Action name must be unique", http.StatusBadRequest)
			return
		}
	}
	actionsMu.Unlock()

	if action.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	if action.Method == "" {
		action.Method = "POST"
	}

	action.ID = generateID()

	actionsMu.Lock()
	actions[action.ID] = action
	actionsMu.Unlock()

	respondJSON(w, action)
	saveToFile()
}

func updateAction(w http.ResponseWriter, r *http.Request) {
	var updatedAction Action
	if err := json.NewDecoder(r.Body).Decode(&updatedAction); err != nil {
		http.Error(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Extract action ID from the URL path
	actionID := strings.TrimPrefix(r.URL.Path, "/actions/")
	if actionID == "" {
		http.Error(w, "Action ID is required", http.StatusBadRequest)
		return
	}

	actionsMu.Lock()
	defer actionsMu.Unlock()

	// Check if the action exists
	action, exists := actions[actionID]
	if !exists {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	// Ensure name uniqueness (excluding self)
	for _, a := range actions {
		if a.ID != actionID && a.Name == updatedAction.Name {
			http.Error(w, "Action name must be unique", http.StatusBadRequest)
			return
		}
	}

	// Apply updates
	action.Name = updatedAction.Name
	action.URL = updatedAction.URL
	action.Method = updatedAction.Method
	action.Headers = updatedAction.Headers
	action.Body = updatedAction.Body
	actions[actionID] = action

	saveToFile() // Persist changes

	respondJSON(w, action) // Send back the updated action
}

func deleteAction(w http.ResponseWriter, r *http.Request) {
	actionID := strings.TrimPrefix(r.URL.Path, "/actions/")           // Extract ID from path
	log.Printf("Received DELETE request for action ID: %s", actionID) // Debugging

	if actionID == "" {
		log.Println("Error: No action ID provided")
		http.Error(w, "Action ID is required", http.StatusBadRequest)
		return
	}

	actionsMu.Lock()
	defer actionsMu.Unlock()

	if _, exists := actions[actionID]; !exists {
		log.Printf("Error: Action ID %s not found", actionID) // Debugging
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	delete(actions, actionID)
	saveToFile() // Persist changes

	log.Printf("Action ID %s deleted successfully", actionID) // Debugging
	w.WriteHeader(http.StatusNoContent)                       // 204 No Content (Success)
}

func triggerAction(w http.ResponseWriter, _ *http.Request, actionID string) {
	actionsMu.Lock()
	action, exists := actions[actionID]
	actionsMu.Unlock()

	if !exists {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	success, logMsg, err := executeAction(action)
	if err != nil {
		http.Error(w, logMsg, http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Action triggered: %s (Success: %v)", logMsg, success)
	}
}

func getLogs(w http.ResponseWriter, _ *http.Request) {
	logsMu.Lock()
	defer logsMu.Unlock()

	respondJSON(w, logs)
}

//
// Utility Functions
//

// executeAction performs the HTTP request defined in the action and logs the result.
func executeAction(action Action) (bool, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(action.Method, action.URL, strings.NewReader(action.Body))
	if err != nil {
		logError(action.ID, "Request creation error: "+err.Error())
		return false, "", err
	}

	for key, value := range action.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	var logMsg string
	success := false

	if err != nil {
		logMsg = "Request failed: " + err.Error()
	} else {
		defer resp.Body.Close()
		logMsg = fmt.Sprintf("Status: %s", resp.Status)
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	logTrigger(action.ID, success, logMsg)
	return success, logMsg, err
}

// respondJSON writes the given data as JSON to the response writer.
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logError appends an error log entry.
func logError(actionID, message string) {
	logsMu.Lock()
	defer logsMu.Unlock()

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		ActionID:  actionID,
		Success:   false,
		Response:  message,
	})
}

// logTrigger appends a log entry for a triggered action and saves data.
func logTrigger(actionID string, success bool, response string) {
	logsMu.Lock()
	defer logsMu.Unlock()

	logs = append(logs, LogEntry{
		Timestamp: time.Now(),
		ActionID:  actionID,
		Success:   success,
		Response:  response,
	})

	saveToFile()
}

// generateID generates a random 16-character hex string as an ID.
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// loadFromFile loads persisted actions and logs from disk.
func loadFromFile() {
	dataMu.Lock()
	defer dataMu.Unlock()

	file, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return // First run, no data yet.
		}
		log.Printf("Error loading data: %v", err)
		return
	}

	var data PersistentData
	if err := json.Unmarshal(file, &data); err != nil {
		log.Printf("Error parsing data: %v", err)
		return
	}

	actions = data.Actions
	logs = data.Logs
}

// saveToFile saves the current actions and logs to disk.
func saveToFile() {
	dataMu.Lock()
	defer dataMu.Unlock()

	data := PersistentData{
		Actions: actions,
		Logs:    logs,
	}

	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return
	}

	if err := os.WriteFile(dataFile, file, 0644); err != nil {
		log.Printf("Error saving data: %v", err)
	}
}

// printHelp prints the descriptive help message.
func printHelp() {
	helpText := `
Companion Proxy - Command Line Interface

Usage:
  companion-proxy <command> [options]

Commands:
  server
      Start the HTTP server.
      Options:
        -port int    Port number for the server (default: 8080)

  add
      Add a new action.
      Options:
        -name string       Unique name for the action (required).
        -url string        URL for the action (required).
        -method string     HTTP method to use (default: POST).
        -header string     Header in "Key:Value" format (optional).
        -body string       Request body (optional).

  edit
      Edit an existing action.
      Options:
        -id string         ID of the action to edit (required).
        -name string       New name for the action (optional).
        -url string        New URL for the action (optional).
        -method string     New HTTP method (optional).
        -header string     Header in "Key:Value" format (optional).
        -body string       New request body (optional).

  delete
      Delete an action.
      Options:
        -id string         ID of the action to delete (required).

  list
      List all actions.

  trigger
      Trigger an action.
      Options:
        -id string         ID of the action to trigger.
        -name string       Name of the action to trigger.
      (Either -id or -name is required.)

  logs
      Display the logs of executed actions.

  help
      Display this help message.

Examples:
  Start the server on the default port (8080):
      companion-proxy server

  Start the server on port 9000:
      companion-proxy server -port 9000

  Add a new action:
      companion-proxy add -name "MyAction" -url "http://example.com/api" -method POST -header "Authorization:Bearer token" -body '{"key": "value"}'

  Edit an existing action:
      companion-proxy edit -id abc123 -name "UpdatedAction" -url "http://example.com/updated"

  Delete an action:
      companion-proxy delete -id abc123

  List all actions:
      companion-proxy list

  Trigger an action by name:
      companion-proxy trigger -name "MyAction"

  Display logs:
      companion-proxy logs
`
	fmt.Println(strings.TrimSpace(helpText))
}
