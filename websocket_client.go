package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func handleWebsocketClientConnection() {
	defer tasksWg.Done()

	configMu.Lock()

	// WebSocket server URL
	serverURL := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("localhost:%d", config.WebsocketPort),
	}

	configMu.Unlock()

	websocketLogger.White("Connnecting to websocket server...")

	// Establish WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(serverURL.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	// Send a message once connected
	go func() {
		time.Sleep(time.Second) // Short delay to ensure connection is established

		sendClientHello(conn)
	}()

	// Listen for incoming messages from the server
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("Error reading message: %v", err))
			break
		}

		onMessage(conn, message)
	}
}

func sendClientHello(conn *websocket.Conn) {
	clientHelloMsg := ClientHelloMessage{
		MonitorType: "SNS",
		TypeName:    "CLIENT_HELLO",
	}
	bytes, err := json.Marshal(clientHelloMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending client hello message: %v", err))
	}

	conn.WriteMessage(websocket.TextMessage, bytes)
}

var onMessage = func(conn *websocket.Conn, message []byte) {
	var messageType MessageType

	err := json.Unmarshal(message, &messageType)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error unmarshalling message type: %s", err))
		return
	}

	switch messageType.TypeName {
	case "ADD":
		var addMessage AddMessage

		err = json.Unmarshal(message, &addMessage)
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("Error unmarshalling add message: %s", err))
			return
		}

		addMessage.AddQuery = strings.TrimSpace(addMessage.AddQuery)
		for strings.Contains(addMessage.AddQuery, "  ") {
			addMessage.AddQuery = strings.Replace(addMessage.AddQuery, "  ", " ", -1)
		}

		err = handleAdd(&addMessage)
		if err != nil {
			if aerr, ok := err.(*AlreadyMonitoredError); ok {
				websocketLogger.Red(fmt.Sprintf("already monitoring %s", aerr.queryValue))

				errText := fmt.Sprintf("Fehler: %s \"%s\" ist bereits im Monitor.", addMessage.InputType, addMessage.AddQuery)
				sendError(conn, addMessage.TaskId, errText)
				return
			} else {
				websocketLogger.Red(fmt.Sprintf("error adding query: %v", err))

				sendError(conn, addMessage.TaskId, "Interner Fehler")
				return
			}
		}

		websocketLogger.Cyan(fmt.Sprintf("Added %s %s", addMessage.InputType, addMessage.AddQuery))

		successText := fmt.Sprintf("%s \"%s\" wurde erfolgreich hinzugefügt.", addMessage.InputType, addMessage.AddQuery)
		sendSuccess(conn, addMessage.TaskId, successText)
	case "REMOVE":
		var removeMessage RemoveMessage

		err = json.Unmarshal(message, &removeMessage)
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("Error unmarshalling remove message: %s", err))
			return
		}

		removeMessage.RemoveQuery = strings.TrimSpace(removeMessage.RemoveQuery)
		for strings.Contains(removeMessage.RemoveQuery, "  ") {
			removeMessage.RemoveQuery = strings.Replace(removeMessage.RemoveQuery, "  ", " ", -1)
		}

		err := handleRemove(&removeMessage)
		if err != nil {
			if aerr, ok := err.(*QueryNotFoundError); ok {
				websocketLogger.Red(aerr)

				errText := fmt.Sprintf("Fehler: %s \"%s\" wurde nicht gefunden.", removeMessage.InputType, removeMessage.RemoveQuery)
				sendError(conn, removeMessage.TaskId, errText)
				return
			} else {
				websocketLogger.Red(fmt.Sprintf("error removing query: %v", err))

				sendError(conn, removeMessage.TaskId, "Interner Fehler.")
				return
			}
		}

		websocketLogger.Cyan(fmt.Sprintf("Removed %s %s", removeMessage.InputType, removeMessage.RemoveQuery))

		successText := fmt.Sprintf("%s \"%s\" wurde erfolgreich gelöscht.", removeMessage.InputType, removeMessage.RemoveQuery)
		sendSuccess(conn, removeMessage.TaskId, successText)
	case "LIST":
		var listMessage ListMessage

		err = json.Unmarshal(message, &listMessage)
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("Error unmarshalling list message: %s", err))
			return
		}

		list, err := handleList(&listMessage)
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("error listing query: %v", err))

			sendError(conn, listMessage.TaskId, "Interner Fehler.")
			return
		}

		successText := fmt.Sprintf("%s Liste:", listMessage.InputType)
		sendSuccessList(conn, listMessage.TaskId, successText, list)
	default:
		websocketLogger.Red(fmt.Sprintf("Unexpected message typename: %s", messageType.TypeName))
		return
	}
}

func sendSuccess(conn *websocket.Conn, taskId string, successText string) {
	successMsg := SuccessResponse{
		TypeName:    "SUCCESS",
		TaskId:      taskId,
		SuccessText: successText,
	}
	bytes, err := json.Marshal(successMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending success message: %v", err))
	}

	conn.WriteMessage(websocket.TextMessage, bytes)
}

func sendSuccessList(conn *websocket.Conn, taskId string, successText string, list []string) {
	successMsg := SuccessListResponse{
		TypeName:    "SUCCESS",
		TaskId:      taskId,
		SuccessText: successText,
		List:        list,
	}
	bytes, err := json.Marshal(successMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending success list message: %v", err))
	}

	conn.WriteMessage(websocket.TextMessage, bytes)
}

func sendError(conn *websocket.Conn, taskId string, errorText string) {
	errMsg := ErrorResponse{
		TypeName:  "ERROR",
		TaskId:    taskId,
		ErrorText: errorText,
	}
	bytes, err := json.Marshal(errMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending error message: %v", err))
	}

	conn.WriteMessage(websocket.TextMessage, bytes)
}
