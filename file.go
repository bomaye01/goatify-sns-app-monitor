package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var fileLogger *log.Logger = nil

const (
	pathConfig        string = "./config.json"
	pathProductStates string = "./product_states.json"
	pathLogfileFolder string = "./logs"
	pathProxyFolder   string = "./proxies"
)

func readConfig() error {
	configMu.Lock()
	defer configMu.Unlock()

	if _, err := os.Stat(pathConfig); os.IsNotExist(err) {
		bytes, err := json.MarshalIndent(defaultConfig, "", "\t")
		if err != nil {
			return fmt.Errorf("create default config: error marshalling config: %v", err)
		}

		err = os.WriteFile(pathConfig, bytes, os.ModePerm)
		if err != nil {
			return fmt.Errorf("create default config: error creating logs folder: %v", err)
		}

		config = &Config{}
		*config = defaultConfig
	} else {
		bytes, err := os.ReadFile(pathConfig)
		if err != nil {
			return fmt.Errorf("error reading config: %v", err)
		}

		var newConfig Config

		err = json.Unmarshal(bytes, &newConfig)
		if err != nil {
			return fmt.Errorf("error unmarshalling config: %v", err)
		}

		config = &newConfig
	}
	return nil
}

func refreshConfig() {
	go func() {
		for {
			readConfig()

			time.Sleep(time.Second * 1)
		}
	}()
}

func readProductStates() (*ProductStates, error) {
	productStateFileMu.Lock()
	defer productStateFileMu.Unlock()

	if _, err := os.Stat(pathProductStates); os.IsNotExist(err) {
		bytes, err := json.MarshalIndent(defaultProductStates, "", "\t")
		if err != nil {
			return nil, fmt.Errorf("create default product states: error marshalling product states: %v", err)
		}

		err = os.WriteFile(pathProductStates, bytes, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("create default product states: error creating logs folder: %v", err)
		}

		productStates := defaultProductStates
		return &productStates, err
	} else {
		bytes, err := os.ReadFile(pathProductStates)
		if err != nil {
			return nil, fmt.Errorf("error reading \"%s\": %v", pathProductStates, err)
		}

		var fileProductStates *ProductStates

		err = json.Unmarshal(bytes, &fileProductStates)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling product states: %v", err)
		}

		return fileProductStates, nil
	}
}

func writeProductStates() {
	if productStates == nil {
		return
	}

	statesNormalMu.Lock()
	statesLoadMu.Lock()
	productStateFileMu.Lock()
	defer productStateFileMu.Unlock()
	defer statesLoadMu.Unlock()
	defer statesNormalMu.Unlock()

	bytes, err := json.MarshalIndent(productStates, "", "\t")
	if err != nil {
		fileSystemLogger.Red(fmt.Sprintf("Error marshalling product states: %v", err))
	}

	err = os.WriteFile(pathProductStates, bytes, 0644)
	if err != nil {
		fileSystemLogger.Red(fmt.Sprintf("Error writing product states: %v", err))
	}
}

func readProxyfile(filename string) ([]*proxy, error) {
	proxyfileMu.Lock()
	defer proxyfileMu.Unlock()

	proxies := []*proxy{}

	if filename == "" {
		return proxies, nil
	}

	filepath := fmt.Sprintf("%s/%s", pathProxyFolder, filename)

	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading \"%s\": %v", filepath, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(bytes)))
	for scanner.Scan() {
		line := scanner.Text()

		// Split the line into its components based on the colon separator.
		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			return nil, fmt.Errorf("invalid proxy format in line: %s", line)
		}

		// Create a proxy struct and append it to the proxies slice.
		proxies = append(proxies, &proxy{
			host:     parts[0],
			port:     parts[1],
			username: parts[2],
			password: parts[3],
		})
	}

	return proxies, nil
}

func writeProxyfile(filename string, proxies []*proxy) {
	proxyfileMu.Lock()

	go func() {
		defer proxyfileMu.Unlock()

		strVal := ""
		for _, proxy := range proxies {
			strVal += fmt.Sprintf("%s:%s:%s:%s\n", proxy.host, proxy.port, proxy.username, proxy.password)
		}
		strVal = strings.TrimSuffix(strVal, "\n")

		err := os.WriteFile(fmt.Sprintf("%s/%s", pathProxyFolder, filename), []byte(strVal), 0644)
		if err != nil {
			fileSystemLogger.Red(fmt.Sprintf("Error writing to proxyfile \"%s\": %v", filename, err))
			return
		}
	}()
}

func checkLogfolder() error {
	if _, err := os.Stat(pathLogfileFolder); os.IsNotExist(err) {
		err := os.Mkdir(pathLogfileFolder, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating logs folder: %v", err)
		}
	}
	return nil
}

func checkProxyfolder() error {
	if _, err := os.Stat(pathProxyFolder); os.IsNotExist(err) {
		err := os.Mkdir(pathProxyFolder, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating proxies folder: %v", err)
		}
	}
	return nil
}

func createLogfile(path string) (*os.File, error) {
	logfile, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating log file: %v", err)
	}
	return logfile, nil
}
