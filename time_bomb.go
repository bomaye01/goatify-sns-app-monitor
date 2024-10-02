package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type TimeResponse struct {
	Datetime string `json:"datetime"`
}

func checkExceededTimeCheckSystemTime() bool {
	terminationDate := time.Date(2024, 10, 9, 0, 0, 0, 0, time.UTC)

	// Get the current system time
	currentTime := time.Now().UTC()

	// Check if the current time is past the termination date
	return currentTime.After(terminationDate)
}

func checkExceededTimeCheckFetch() bool {
	terminationDate := time.Date(2024, 10, 9, 0, 0, 0, 0, time.UTC)

	resp, err := http.Get("https://worldtimeapi.org/api/timezone/Etc/UTC")
	if err != nil {
		log.Printf("Time check failed (1): %v. Exiting...\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Time check failed (2): %v. Exiting...\n", err)
		return false
	}

	// Parse the JSON response
	var timeResponse TimeResponse
	if err := json.Unmarshal(body, &timeResponse); err != nil {
		log.Printf("Time check failed (3): %v. Exiting...\n", err)
		return false
	}

	serverTime, err := time.Parse(time.RFC3339, timeResponse.Datetime)
	if err != nil {
		log.Printf("Time check failed (4): %v. Exiting...\n", err)
		return false
	}

	// Check if the current time is past the termination date
	return serverTime.After(terminationDate)
}
