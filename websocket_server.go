package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	clientId string
	conn     *websocket.Conn
}

func startWebsocketServer() {
	defer tasksWg.Done()

	configMu.Lock()
	http.HandleFunc(fmt.Sprintf("/goatify-monitor-control-%s", config.WebsocketPathSuffix), handleWebSocket)

	// Start the server
	port := strconv.Itoa(config.WebsocketPort)
	configMu.Unlock()

	websocketLogger.White(fmt.Sprintf("Starting server on port %s", port))
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error upgrading to WebSocket: %v", err))
		return
	}
	client := &Client{
		conn: conn,
	}
	defer func() {
		onDisconnect(client)
		conn.Close()
	}()

	// Trigger onConnect event
	onConnect(client)

	// Listen for messages from the client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			websocketLogger.Red(fmt.Sprintf("Error reading message: %v", err))
			break
		}
		// Trigger onMessage event
		onMessage(client, message)
	}
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Event handlers
var onConnect = func(client *Client) {
	client.clientId = uuid.NewString()

	websocketLogger.Cyan(fmt.Sprintf("[%s] New client connected", client.clientId))
}

var onMessage = func(client *Client, message []byte) {
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
				sendError(client, addMessage.TaskId, errText)
				return
			} else if nerr, ok := err.(*NotAProductPageError); ok {
				websocketLogger.Red(fmt.Sprintf("not a product page %s", nerr.url))

				errText := fmt.Sprintf("Fehler: Ungültige Produkt-Url: %s", addMessage.AddQuery)
				sendError(client, addMessage.TaskId, errText)
				return
			} else if cerr, ok := err.(*ProductPageCheckError); ok {
				websocketLogger.Red(fmt.Sprintf("error checking %s", cerr.url))

				errText := fmt.Sprintf("Fehler aufgetreten beim Prüfen von %s \"%s\"", addMessage.InputType, addMessage.AddQuery)
				sendError(client, addMessage.TaskId, errText)
				return
			} else {
				websocketLogger.Red(fmt.Sprintf("error adding query: %v", err))

				sendError(client, addMessage.TaskId, "Interner Fehler")
				return
			}
		}

		websocketLogger.Cyan(fmt.Sprintf("Added %s %s", addMessage.InputType, addMessage.AddQuery))

		successText := fmt.Sprintf("%s \"%s\" wurde erfolgreich hinzugefügt.", addMessage.InputType, addMessage.AddQuery)
		sendSuccess(client, addMessage.TaskId, successText)
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
				sendError(client, removeMessage.TaskId, errText)
				return
			} else {
				websocketLogger.Red(fmt.Sprintf("error removing query: %v", err))

				sendError(client, removeMessage.TaskId, "Interner Fehler.")
				return
			}
		}

		websocketLogger.Cyan(fmt.Sprintf("Removed %s %s", removeMessage.InputType, removeMessage.RemoveQuery))

		successText := fmt.Sprintf("%s \"%s\" wurde erfolgreich gelöscht.", removeMessage.InputType, removeMessage.RemoveQuery)
		sendSuccess(client, removeMessage.TaskId, successText)
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

			sendError(client, listMessage.TaskId, "Interner Fehler.")
			return
		}

		successText := fmt.Sprintf("%s Liste:", listMessage.InputType)
		sendSuccessList(client, listMessage.TaskId, successText, list)
	default:
		websocketLogger.Red(fmt.Sprintf("Unexpected message typename: %s", messageType.TypeName))
		return
	}
}

var onDisconnect = func(client *Client) {
	websocketLogger.Cyan(fmt.Sprintf("[%s]: Client disconnected", client.clientId))
}

func sendSuccess(client *Client, taskId string, successText string) {
	errMsg := SuccessResponse{
		TypeName:    "SUCCESS",
		TaskId:      taskId,
		SuccessTest: successText,
	}
	bytes, err := json.Marshal(errMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending success message: %v", err))
	}

	client.conn.WriteMessage(websocket.TextMessage, bytes)
}

func sendSuccessList(client *Client, taskId string, successText string, list []string) {
	errMsg := SuccessListResponse{
		TypeName:    "SUCCESS",
		TaskId:      taskId,
		SuccessTest: successText,
		List:        list,
	}
	bytes, err := json.Marshal(errMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending success list message: %v", err))
	}

	client.conn.WriteMessage(websocket.TextMessage, bytes)
}

func sendError(client *Client, taskId string, errorText string) {
	errMsg := ErrorResponse{
		TypeName:  "ERROR",
		TaskId:    taskId,
		ErrorText: errorText,
	}
	bytes, err := json.Marshal(errMsg)
	if err != nil {
		websocketLogger.Red(fmt.Sprintf("Error sending error message: %v", err))
	}

	client.conn.WriteMessage(websocket.TextMessage, bytes)
}
