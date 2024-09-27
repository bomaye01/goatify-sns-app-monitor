package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	VERSION = "0.3.3"
)

// Lock order as below. Before that, take the individual handler lock
var configMu sync.RWMutex = sync.RWMutex{}
var statesNormalMu sync.Mutex = sync.Mutex{}
var statesLoadMu sync.Mutex = sync.Mutex{}
var proxyfileMu sync.Mutex = sync.Mutex{}
var productStateFileMu sync.Mutex = sync.Mutex{}

var tasksWg sync.WaitGroup = sync.WaitGroup{}

// Logger
var mainLogger *Logger = NewLogger("MAIN")
var fileSystemLogger *Logger = NewLogger("FILE")
var websocketLogger *Logger = NewLogger("WEBSOCKET")

var config *Config = nil

var proxyHandler *ProxyHandler = nil
var webhookHandler *WebhookHandler = nil

var normalTaskGroup *NormalTaskGroup = nil
var loadTaskGroup *LoadTaskGroup = nil

var fileLoggingEnabled bool = true

func main() {
	err := checkLogfolder()
	if err != nil {
		log.Printf("Init: %v", err)
		return
	}
	err = checkProxyfolder()
	if err != nil {
		log.Printf("Init: %v", err)
		return
	}

	// Logfile setup
	logfile, err := initLogfile()
	if err != nil {
		log.Printf("Init: Error setting up logfile: %v\n", err)
		return
	}
	defer logfile.Close()

	// file logger setup
	fileLogger = log.New(logfile, "", log.LstdFlags|log.Lshortfile)
	if fileLogger == nil {
		log.Println("Init: Error creating file logger")
		return
	}

	// Load config
	err = readConfig()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Init: %v", err))
		return
	}
	refreshConfig()

	initTerminal()

	tasksWg.Add(1)
	go handleWebsocketClientConnection()

	// Load product states
	productStates, err = readProductStates()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Init: %v", err))
		return
	}
	if productStates == nil {
		productStates = &ProductStates{}
	}

	formatProductStates()

	configMu.RLock()

	// Check file logging
	if !config.EnableFileLogging {
		fileLogger.Println("file logging disabled")
		fileLoggingEnabled = false
	}

	// Load proxies
	proxies, err := readProxyfile(config.ProxyfileName)
	if err != nil {
		configMu.RUnlock()

		mainLogger.Red(fmt.Sprintf("Init: %v", err))
		return
	}
	configMu.RUnlock()

	// Create handlers
	proxyHandler = NewProxyHandler(proxies)
	webhookHandler = NewWebhookHandler()

	configMu.RLock()

	mainLogger.White(fmt.Sprintf("Starting SNS monitor (v%s) ...", VERSION))

	// Create task groups
	statesNormalMu.Lock()
	statesLoadMu.Lock()

	normalSkus := NormalGetAllSkus()

	normalTaskGroup, err = NewNormalTaskGroup(proxyHandler, webhookHandler, normalSkus)
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error creating normal task group: %v", err))
		return
	}

	loadTaskGroup, err = NewLoadTaskGroup(proxyHandler, webhookHandler, productStates.Load.LastKnownPid, productStates.Load.KeywordQueries)
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error creating normal task group: %v", err))
		return
	}

	// Associate task groups with one another
	normalTaskGroup.LinkToLoadTaskGroup(loadTaskGroup)
	loadTaskGroup.LinkToNormalTaskGroup(normalTaskGroup)

	// Create normal tasks
	for i := range config.NormalTask.NumTasks {
		tasksWg.Add(1)

		taskName := fmt.Sprintf("NORMAL: %02d", i)
		normalTask, err := NewNormalTask(taskName, normalTaskGroup)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error creating initial normal task %s: %v", taskName, err))
			return
		}

		err = normalTaskGroup.AddTask(normalTask.BaseTask)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error adding normal task %s to task group: %v", taskName, err))
			return
		}
	}

	// Create load tasks
	for i := range config.LoadTask.NumTasks {
		tasksWg.Add(1)

		taskName := fmt.Sprintf("LOAD: %02d", i)
		loadTask, err := NewLoadTask(taskName, loadTaskGroup)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error creating initial load task %s: %v", taskName, err))
			return
		}

		err = loadTaskGroup.AddTask(loadTask.BaseTask)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error adding load task %s to task group: %v", taskName, err))
			return
		}
	}

	statesNormalMu.Unlock()
	statesLoadMu.Unlock()

	// Start webhook handler
	webhookHandler.Start()

	// Launch tasks
	err = normalTaskGroup.StartAllTasks()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error starting normal tasks: %v", err))
		return
	}
	defer normalTaskGroup.StopAllTasks()

	err = loadTaskGroup.StartAllTasks()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error starting load tasks: %v", err))
		return
	}
	defer loadTaskGroup.StopAllTasks()

	configMu.RUnlock()

	tasksWg.Wait()

	webhookHandler.Stop()
}

func initTerminal() {
	cmd := exec.Command("cmd", "/c", "cls") // Windows only
	cmd.Stdout = os.Stdout
	cmd.Run()

	enableVirtualTerminalProcessing()

	log.SetFlags(0)

	configMu.RLock()
	log.Printf("\033]0;SNS Monitor (v%s) - linus - %s\007", VERSION, config.InstanceName)
	configMu.RUnlock()
}

func initLogfile() (*os.File, error) {
	logfileName := "log_" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	path := fmt.Sprintf("%s/%s.txt", pathLogfileFolder, logfileName)

	logfile, err := createLogfile(path)
	if err != nil {
		return nil, fmt.Errorf("error creating log file: %v", err)
	}

	return logfile, nil
}

func enableVirtualTerminalProcessing() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")

	var mode uint32
	handle := syscall.Handle(os.Stdout.Fd())
	getConsoleMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	mode |= 0x0004
	setConsoleMode.Call(uintptr(handle), uintptr(mode))
}

func formatProductStates() {
	statesNormalMu.Lock()
	defer statesNormalMu.Unlock()
	statesLoadMu.Lock()
	defer statesLoadMu.Unlock()

	for i, state := range productStates.Normal.ProductStates {
		productStates.Normal.ProductStates[i].Sku = strings.ToUpper(strings.TrimSpace(state.Sku))
	}

	productStates.Load.LastKnownPid = strings.ToUpper(strings.TrimSpace(productStates.Load.LastKnownPid))

	for i, query := range productStates.Load.KeywordQueries {
		productStates.Load.KeywordQueries[i] = strings.ToLower(strings.TrimSpace(query))
	}

	for i, notified := range productStates.Load.NotifiedProducts {
		productStates.Load.NotifiedProducts[i].Sku = strings.ToUpper(strings.TrimSpace(notified.Sku))
	}
}
