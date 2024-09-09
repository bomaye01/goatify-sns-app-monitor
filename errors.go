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

type NotAProductPageError struct {
	url string
}

func (e *NotAProductPageError) Error() string {
	return fmt.Sprintf("url \"%s\" is not a value product page", e.url)
}

type ProductPageCheckError struct {
	err error
	url string
}

func (e *ProductPageCheckError) Error() string {
	return fmt.Sprintf("product page check failed on \"%s\": %v", e.url, e.err)
}
