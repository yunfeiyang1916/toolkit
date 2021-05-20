package sql

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	workingDir     = "/"
	stackCache     map[string]string
	stackCacheLock sync.RWMutex
)

func init() {
	wd, err := os.Getwd()
	if err == nil {
		workingDir = filepath.ToSlash(wd) + "/"
	}
	stackCache = make(map[string]string)
}

func funcName(source string) string {
	name, err := extractCallerInfo(source)
	if err != nil {
		return ""
	}

	return name
}

func callerFuncSkip(source string) (uintptr, error) {
	callerInfo := strings.Split(source, ":")
	if len(callerInfo) < 2 {
		return 0, errors.New("error during runtime.Callers")
	}
	file := callerInfo[0]
	line, err := strconv.Atoi(callerInfo[1])
	if err != nil {
		return 0, err
	}
	for i := 2; i < 15; i++ {
		pc, f, l, ok := runtime.Caller(i)
		if ok && f == file && l == line {
			return pc, nil
		}
	}
	return 0, errors.New("error during runtime.Callers")
}

func extractCallerInfo(source string) (string, error) {
	stackCacheLock.RLock()
	ctx, ok := stackCache[source]
	stackCacheLock.RUnlock()
	if ok {
		return ctx, nil
	}

	pc, err := callerFuncSkip(source)
	if err != nil {
		return "", err
	}

	// look up the details of the given caller
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return "", errors.New("error during runtime.FuncForPC")
	}

	funcName := funcInfo.Name()
	if strings.HasPrefix(funcName, workingDir) {
		funcName = funcName[len(workingDir):]
	}
	funcName = funcName[strings.LastIndex(funcName, "/")+1:]

	stackCacheLock.Lock()
	stackCache[source] = funcName
	stackCacheLock.Unlock()
	return funcName, nil
}
