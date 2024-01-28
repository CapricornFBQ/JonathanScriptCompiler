package parser

import (
	"log"
	"runtime"
	"strings"
	"time"
)

var globalStartPc int = 0

// colorRed colorGreen colorYellow
var colorList = [8]string{"\033[33m", "\033[34m", "\033[35m" /*"\033[31m",*/, "\033[36m", "\033[37m", "\033[32m", "\033[38m"}

const colorReset = "\033[0m"

// trace get the name and the stack depth
func trace(funcName string) (string, int, time.Time, int) {
	start := time.Now()
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		// get the depth
		depth := 0
		pc := make([]uintptr, 10) // at most 10 layers deep
		n := runtime.Callers(0, pc)
		if globalStartPc == 0 {
			globalStartPc = n
		}
		depth = n - globalStartPc
		// random
		//colorIndex := rand.Intn(8)
		var colorIndex = depth % 8
		// indent
		indent := strings.Repeat("  ", depth*2)
		log.Printf("%s%s start: %s %s", colorList[colorIndex], indent, funcName, colorReset)
		return funcName, depth, start, colorIndex
	}
	return funcName, 0, start, 0
}

// untrace print info
func untrace(funcName string, depth int, start time.Time, colorIndex int) {
	indent := strings.Repeat("  ", depth*2)
	log.Printf("%s%s end  : %s，duration: %s %s", colorList[colorIndex], indent, funcName, time.Since(start), colorReset)
}
