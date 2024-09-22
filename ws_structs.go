package main

// Websocket receive structs
type MessageType struct {
	TypeName string `json:"typeName"`
}

type AddMessage struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	InputType string `json:"inputType"`
	AddQuery  string `json:"addQuery"`
}

type RemoveMessage struct {
	TypeName    string `json:"typeName"`
	TaskId      string `json:"taskId"`
	InputType   string `json:"inputType"`
	RemoveQuery string `json:"removeQuery"`
}

type ListMessage struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	InputType string `json:"inputType"`
}

// Websocket send structs

type SuccessResponse struct {
	TypeName    string `json:"typeName"`
	TaskId      string `json:"taskId"`
	SuccessText string `json:"successText"`
}

type SuccessListResponse struct {
	TypeName    string   `json:"typeName"`
	TaskId      string   `json:"taskId"`
	SuccessText string   `json:"successText"`
	List        []string `json:"list"`
}

type ErrorResponse struct {
	TypeName  string `json:"typeName"`
	TaskId    string `json:"taskId"`
	ErrorText string `json:"errorText"`
}

type ClientHelloMessage struct {
	TypeName    string `json:"typeName"`
	MonitorType string `json:"monitorType"`
}
