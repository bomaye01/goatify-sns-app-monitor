package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Lock order as below. After that, take the individual handler lock
var configMu sync.RWMutex = sync.RWMutex{}
var proxyfileMu sync.Mutex = sync.Mutex{}
var productStateFileMu sync.Mutex = sync.Mutex{}
var statesNormalMu sync.Mutex = sync.Mutex{}
var statesLoadMu sync.Mutex = sync.Mutex{}
var taskReferenceMu sync.Mutex = sync.Mutex{}
var taskCountMu sync.Mutex = sync.Mutex{}

var tasksWg sync.WaitGroup = sync.WaitGroup{}

// Logger
var mainLogger *Logger = NewLogger("MAIN")
var fileSystemLogger *Logger = NewLogger("FILE")
var websocketLogger *Logger = NewLogger("WEBSOCKET")

var config *Config = nil
var productStates *ProductStates = nil

var normalTasksByProductUrl map[string][]*NormalTask = make(map[string][]*NormalTask)
var loadTasks []*LoadTask

var proxyHandler *ProxyHandler = nil
var webhookHandler *WebhookHandler = nil

var normalTaskCount int = 0

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

	initTerminal()

	tasksWg.Add(1)
	go startWebsocketServer()

	mainLogger.White("Starting SNS monitor...")

	// Load config
	err = readConfig()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Init: %v", err))
		return
	}
	refreshConfig()

	// Load product states
	productStates, err = readProductStates()
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Init: %v", err))
		return
	}
	if productStates == nil {
		productStates = &ProductStates{}
	}

	// Load proxies
	configMu.RLock()

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

	// Create task groups
	statesNormalMu.Lock()
	statesLoadMu.Lock()

	normalTaskGroup, err := NewNormalTaskGroup(proxyHandler, webhookHandler, productStates.Normal.SKUs)
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error creating normal task group: %v", err))
		return
	}

	loadTaskGroup, err := NewLoadTaskGroup(proxyHandler, webhookHandler, productStates.Load.LastKnownPid, productStates.Load.SkuQueries, productStates.Load.KeywordQueries)
	if err != nil {
		mainLogger.Red(fmt.Sprintf("Error creating normal task group: %v", err))
		return
	}

	// Create normal tasks
	for i := range config.NormalTask.NumTasks {
		tasksWg.Add(1)

		taskName := fmt.Sprintf("NORMAL: %02d", i)
		normalTask, err := NewNormalTask(taskName)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error creating initial normal task %s: %v", taskName, err))
			return
		}

		err = normalTaskGroup.AddTask(normalTask)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error adding normal task %s to task group: %v", taskName, err))
			return
		}
	}

	// Create load tasks
	for i := range config.LoadTask.NumTasks {
		tasksWg.Add(1)

		taskName := fmt.Sprintf("NORMAL: %02d", i)
		loadTask, err := NewLoadTask(taskName)
		if err != nil {
			mainLogger.Red(fmt.Sprintf("Error creating initial load task %s: %v", taskName, err))
			return
		}

		err = loadTaskGroup.AddTask(loadTask)
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
	log.Print("\033]0;SNS Monitor - linus\007")
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
