package main

// Cloud build agent, he will be responsible for the execution of every command inside of the environment and for the communication with the server to update the environment status
// The agent will be a go routine that will be running inside of the environment instance and will be responsible for the execution of every command inside of the environment and for the communication with the server to update the environment status
// Will be established a connection with the server and the agent will be waiting for commands to execute
// The agent will be responsible for the execution of the commands and for the communication with the server to update the environment status

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type Command struct {
	Command string `json:"command"`
}

func main() {
	listenPort := "80"

	// Start listening on port 80 for incoming connections
	listener, err := net.Listen("tcp", ":"+listenPort)
	if err != nil {
		log.Fatalf("Failed to start listener on port %s: %s", listenPort, err)
	}
	defer listener.Close()
	log.Printf("Agent is listening on port %s", listenPort)

	// Wait for incoming connections and handle them concurrently
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %s", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the incoming command
	commandBytes, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		log.Printf("Failed to read command: %s", err)
		return
	}

	// Unmarshal the command JSON
	var command Command
	err = json.Unmarshal(commandBytes, &command)
	if err != nil {
		log.Printf("Failed to unmarshal command: %s", err)
		return
	}

	// Execute the command inside the environment
	output, err := executeCommand(command.Command)
	if err != nil {
		log.Printf("Failed to execute command '%s': %s", command.Command, err)
		return
	}

	// Prepare the response with the command output
	response := struct {
		Command string `json:"command"`
		Output  string `json:"output"`
	}{
		Command: command.Command,
		Output:  output,
	}

	// Marshal the response to JSON
	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %s", err)
		return
	}

	// Send the response back to the client
	_, err = conn.Write(responseBytes)
	if err != nil {
		log.Printf("Failed to send response: %s", err)
		return
	}
}

func executeCommand(command string) (string, error) {
	// Split the command into the command and its arguments
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)

	// Set up the command's output and error pipes
	cmdOutput, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %s", err)
	}
	cmdError, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %s", err)
	}

	// Start the command
	err = cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to start command: %s", err)
	}

	// Read the output and error asynchronously
	var wg sync.WaitGroup
	wg.Add(2)

	var output, errorOutput string

	go func() {
		defer wg.Done()
		output, _ = readAllLines(cmdOutput)
	}()

	go func() {
		defer wg.Done()
		errorOutput, _ = readAllLines(cmdError)
	}()

	// Wait for the command to finish and get the exit status
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		return "", fmt.Errorf("command failed: %s", err)
	}

	// Combine the output and error output
	result := output + errorOutput
	return result, nil
}

func readAllLines(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}
