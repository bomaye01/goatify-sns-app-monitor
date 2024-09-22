package main

import "fmt"

type TaskNotReadyError struct{}

func (e *TaskNotReadyError) Error() string {
	return "task not ready"
}

type TaskRunningError struct{}

func (e *TaskRunningError) Error() string {
	return "task still running"
}

type RequestError struct {
	location      string
	err           error
	proxyAsString string
}

func (e *RequestError) Error() string {
	if e.proxyAsString != "" {
		return fmt.Sprintf("%s: request failed: %v Proxy: %s", e.location, e.err, e.proxyAsString)
	}
	return fmt.Sprintf("%s: request failed: %v", e.location, e.err)
}

type StatusCodeError struct {
	location      string
	statusCode    int
	statusText    string
	proxyAsString string
}

func (e *StatusCodeError) Error() string {
	if e.proxyAsString != "" {
		return fmt.Sprintf("%s: request failed with status code %s. Proxy: %s", e.location, e.statusText, e.proxyAsString)
	}
	return fmt.Sprintf("%s: request failed with status code %s", e.location, e.statusText)
}

type AlreadyMonitoredError struct {
	queryType  string
	queryValue string
}

func (e *AlreadyMonitoredError) Error() string {
	return fmt.Sprintf("%s \"%s\" is already being monitored", e.queryType, e.queryValue)
}

type QueryNotFoundError struct {
	queryType  string
	queryValue string
}

func (e *QueryNotFoundError) Error() string {
	return fmt.Sprintf("%s \"%s\" not found", e.queryType, e.queryValue)
}

type AlreadyIncludedError struct {
	statesType    string
	includedType  string
	includedValue string
}

func (e *AlreadyIncludedError) Error() string {
	return fmt.Sprintf("%s \"%s\" already included in %s product states", e.includedType, e.includedValue, e.statesType)
}

type NotIncludedError struct {
	statesType    string
	includedType  string
	includedValue string
}

func (e *NotIncludedError) Error() string {
	return fmt.Sprintf("%s \"%s\" not included in %s product states", e.includedType, e.includedValue, e.statesType)
}
