package main

import (
	"fmt"
	"github.com/hpcloud/tail"
	"net/http"
	"os"
	"os/user"
	"strconv"
	"sync"
)

var logMap = make(map[string]string)
var mu = &sync.Mutex{}
var quitServer = make(chan bool, 1)
var quitIndexer = make(chan bool, 1)
var quitPersist = make(chan bool, 1)
var quitEvent = make(chan bool, 1)

const (
	server         = "server"
	indexer        = "indexer"
	persist        = "persist"
	event          = "event"
	serverLogPath  = "/.xray/logs/xray_server.log"
	indexerLogPath = "/.xray/logs/xray_indexer.log"
	eventLogPath   = "/.xray/logs/xray_event.log"
	persistLogPath = "/.xray/logs/xray_persist.log"
	serverTag      = "[XRAY-SERVER] "
	indexerTag     = "[XRAY-INDEXER] "
	persistTag     = "[XRAY-PERSIST] "
	eventTag       = "[XRAY-EVENT] "
)

func main() {
	http.HandleFunc("/log", logData)
	fmt.Println("serving on http://localhost:8000/log")
	http.ListenAndServe(":8080", nil)
}

type LoggerProp struct {
	StartLogging bool
	Name         string
	Path         string
	LogTag       string
	logChan      chan bool
}

func logData(w http.ResponseWriter, req *http.Request) {
	var slp *LoggerProp
	var ilp *LoggerProp
	var elp *LoggerProp
	var plp *LoggerProp
	logProps := make([]*LoggerProp, 0)
	logProps = PrepareLogSetting(req, slp, logProps, ilp, elp, plp)
	// used for testing
	StartLogging(logProps)
}
func PrepareLogSetting(req *http.Request, slp *LoggerProp, logProps []*LoggerProp, ilp *LoggerProp, elp *LoggerProp, plp *LoggerProp) []*LoggerProp {
	userPath, err := user.Current()
	if err != nil {
		fmt.Print("Failed to locate login user")
	}
	if value := req.URL.Query().Get(server); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			slp = &LoggerProp{StartLogging: hasValue, Name: server, Path: userPath.HomeDir + serverLogPath, LogTag: serverTag, logChan: quitServer}
			logProps = append(logProps, slp)
		}
	}
	if value := req.URL.Query().Get(indexer); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			ilp = &LoggerProp{StartLogging: hasValue, Name: indexer, Path: userPath.HomeDir + indexerLogPath, LogTag: indexerTag, logChan: quitIndexer}
			logProps = append(logProps, ilp)
		}
	}
	if value := req.URL.Query().Get(persist); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			elp = &LoggerProp{StartLogging: hasValue, Name: persist, Path: userPath.HomeDir + persistLogPath, LogTag: persistTag, logChan: quitPersist}
			logProps = append(logProps, elp)
		}
	}
	if value := req.URL.Query().Get(event); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			plp = &LoggerProp{StartLogging: hasValue, Name: event, Path: userPath.HomeDir + eventLogPath, LogTag: eventTag, logChan: quitEvent}
			logProps = append(logProps, plp)
		}
	}
	return logProps
}
func StartLogging(logProps []*LoggerProp) {
	for _, lProp := range logProps {
		if lProp.StartLogging {
			if added := addToMapIfNotExist(lProp.Name); added {
				go TailLogs(lProp.LogTag, lProp.Path, lProp.logChan)
			}
		} else {
			removeFromMap(lProp.Name)
			lProp.logChan <- true
		}
	}
}

// add log name to cache if it not exist
func addToMapIfNotExist(logName string) bool {
	exist := false
	if len(logMap[logName]) == 0 {
		mu.Lock()
		if len(logMap[logName]) == 0 {
			logMap[logName] = logName
			exist = true
		}
		mu.Unlock()
	}
	return exist
}

// remove log name from cache if it not exist
func removeFromMap(logName string) {
	if len(logMap[logName]) > 0 {
		mu.Lock()
		if len(logMap[logName]) > 0 {
			delete(logMap, logName)
		}
		mu.Unlock()
	}
}

func TailLogs(logName, logPath string, quit chan bool) {
	t, _ := tail.TailFile(logPath, tail.Config{
		Follow: true, ReOpen: true, Poll: true, Location: &tail.SeekInfo{Offset: 0, Whence: os.SEEK_END}})
	for line := range t.Lines {
		select {
		case <-quit:
			t.Stop()
			return
		default:
			fmt.Println(logName + line.Text)
		}
	}
}
