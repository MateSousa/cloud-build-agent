package main

// Cloud build agent, he will be responsible for the execution of every command inside of the environment and for the communication with the server to update the environment status
// The agent will be a go routine that will be running inside of the environment instance and will be responsible for the execution of every command inside of the environment and for the communication with the server to update the environment status
// Will be established a websocket connection between the agent and the server backend to allow the server to send commands to the agent and to allow the agent to send the environment status to the server
// The agent will be responsible for the execution of every command inside of the environment and for the communication with the server to update the environment status
// He will listen for commands from the server and execute them inside of the environment

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/gorilla/websocket"
)

var (
	commandChannel = make(chan string)
	statusChannel  = make(chan string)
	upgrader       = websocket.Upgrader{}
)

func main() {
	// establish the websocket connection
	conn, err := establishWebSocketConnection()
	if err != nil {
		log.Fatal("Failed to establish websocket connection:", err)
	}

	// listen for commands from the server
	go listenForCommands(conn)

	// execute the commands inside the environment
	go executeCommands()

	// send the environment status to the server
	go sendStatusUpdates(conn)

	// create a file log in the root to indicate that the agent is running
	_, err = exec.Command("/bin/sh", "-c", "touch /agent.log").CombinedOutput()
	if err != nil {
		log.Println("Failed to create agent log:", err)
	}

	// keep the agent running
	select {}
}

func establishWebSocketConnection() (*websocket.Conn, error) {
	http.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Failed to upgrade connection to WebSocket:", err)
			return
		}
		defer conn.Close()

		// Handle incoming messages from the client
		for {
			_, commandBytes, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read command:", err)
				return
			}

			command := string(commandBytes)
			commandChannel <- command
		}
	})

	// Replace the port number with the desired port
	port := "8080"
	log.Println("Starting WebSocket server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		return nil, err
	}

	// This code will never be reached since ListenAndServe is blocking.
	// Return an error just for consistency.
	return nil, fmt.Errorf("WebSocket server terminated unexpectedly")
}

func listenForCommands(conn *websocket.Conn) {
	for {
		_, commandBytes, err := conn.ReadMessage()
		if err != nil {
			log.Println("Failed to read command:", err)
			continue
		}

		command := string(commandBytes)
		commandChannel <- command
	}
}

func executeCommands() {
	for command := range commandChannel {
		// Execute the command inside the environment
		cmd := exec.Command("/bin/sh", "-c", command)
		output, err := cmd.CombinedOutput()

		// Prepare the response with the command output
		response := struct {
			Command string `json:"command"`
			Output  string `json:"output"`
			Error   string `json:"error"`
		}{
			Command: command,
			Output:  string(output),
			Error:   "",
		}
		if err != nil {
			response.Error = err.Error()
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Println("Failed to marshal response:", err)
			continue
		}

		// Send the command execution response to the status channel
		statusChannel <- string(responseBytes)
	}
}

func sendStatusUpdates(conn *websocket.Conn) {
	for status := range statusChannel {
		// Send the environment status update to the server
		err := conn.WriteMessage(websocket.TextMessage, []byte(status))
		if err != nil {
			log.Println("Failed to send status update:", err)
		}
	}
}
