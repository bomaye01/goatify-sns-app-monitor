package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type TimeResponse struct {
	Datetime string `json:"datetime"`
}

func checkExceededTimeCheckSystemTime() bool {
	terminationDate := time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC)

	// Get the current system time
	currentTime := time.Now().UTC()

	// Check if the current time is past the termination date
	return currentTime.After(terminationDate)
}

func runTimeBomb() {
	consecutiveErrThreshold := 5 // Abort when this number of consecutive fails is hit
	consecutiveErrCount := 0
	retryTimeoutInSeconds := 5

	go func() {
		for {
			if consecutiveErrCount >= consecutiveErrThreshold {
				log.Println("Time check has failed too often")
				os.Exit(0)
			}

			exceeded, err := checkExceededTimeCheckFetch()
			if err != nil {
				log.Printf("Error checking application expiration (%d): %v\n", consecutiveErrCount, err)
				consecutiveErrCount += 1
			} else if exceeded {
				log.Println("Application has expired")
				os.Exit(0)
			} else {
				consecutiveErrCount = 0
			}

			time.Sleep(time.Second * time.Duration(retryTimeoutInSeconds))
		}
	}()
}

func checkExceededTimeCheckFetch() (bool, error) {
	terminationDate := time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC)

	resp, err := http.Get("https://worldtimeapi.org/api/timezone/Etc/UTC")
	if err != nil {
		log.Printf("Time check failed (1): %v. Exiting...\n", err)
		return false, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Time check failed (2): %v. Exiting...\n", err)
		return false, err
	}

	// Parse the JSON response
	var timeResponse TimeResponse
	if err := json.Unmarshal(body, &timeResponse); err != nil {
		log.Printf("Time check failed (3): %v. Exiting...\n", err)
		return false, err
	}

	serverTime, err := time.Parse(time.RFC3339, timeResponse.Datetime)
	if err != nil {
		log.Printf("Time check failed (4): %v. Exiting...\n", err)
		return false, err
	}

	// Check if the current time is past the termination date
	return serverTime.After(terminationDate), nil
}
