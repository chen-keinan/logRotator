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
var quitm1 = make(chan bool, 1)
var quitm2 = make(chan bool, 1)
var quitm3 = make(chan bool, 1)
var quitm4 = make(chan bool, 1)

const (
	ms1          = "ms1"
	ms2          = "ms2"
	ms3          = "ms3"
	ms4          = "ms4"
	ms1LogPath   = "/ms1.log"
	ms2LogPath   = "/ms2.log"
	ms3LogPath   = "/ms2.log"
	ms4LogPath   = "/ms4.log"
	ms1Tag       = "[MS-1] "
	ms2Tag       =  "[MS-1] "
	ms3Tag       = "[MS-1] "
	ms4Tag       = "[MS-1] "
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
	if value := req.URL.Query().Get(ms1); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			slp = &LoggerProp{StartLogging: hasValue, Name: ms1, Path: userPath.HomeDir + ms1LogPath, LogTag: ms1Tag, logChan: quitm1}
			logProps = append(logProps, slp)
		}
	}
	if value := req.URL.Query().Get(ms2); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			ilp = &LoggerProp{StartLogging: hasValue, Name: ms2, Path: userPath.HomeDir + ms2LogPath, LogTag: ms2Tag, logChan: quitm2}
			logProps = append(logProps, ilp)
		}
	}
	if value := req.URL.Query().Get(ms3); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			elp = &LoggerProp{StartLogging: hasValue, Name: ms3, Path: userPath.HomeDir + ms3LogPath, LogTag: ms3Tag, logChan: quitm3}
			logProps = append(logProps, elp)
		}
	}
	if value := req.URL.Query().Get(ms4); len(value) > 0 {
		if hasValue, err := strconv.ParseBool(value); err == nil {
			plp = &LoggerProp{StartLogging: hasValue, Name: ms4, Path: userPath.HomeDir + ms4LogPath, LogTag: ms4Tag, logChan: quitm4}
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
