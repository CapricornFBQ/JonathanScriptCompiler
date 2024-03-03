package parser

import (
	"fmt"
	"jonathan/ast"
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
func trace(funcName string, parser *Parser) (string, int, time.Time, int, *Parser) {
	start := time.Now()
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		// get the depth
		depth := 0
		pc := make([]uintptr, 20) // at most 20 layers deep
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
		log.Printf("%s%s start:[ %s ], current token literal:[ %s ] %s",
			colorList[colorIndex], indent, funcName, parser.curToken.Literal, colorReset)
		return funcName, depth, start, colorIndex, parser
	}
	return funcName, 0, start, 0, parser
}

// untrace print info
func untrace(funcName string, depth int, start time.Time, colorIndex int, parser *Parser) {
	indent := strings.Repeat("  ", depth*2)
	log.Printf("%s%s end  :[ %s ], current token literal:[ %s ] ，duration:[ %s ] %s",
		colorList[colorIndex], indent, funcName, parser.curToken.Literal, time.Since(start), colorReset)
	PrintStatments(parser.currentParsedStatements)
}

// PrintStatments
func PrintStatments(statements []ast.Statement) {
	if len(statements) == 0 {
		fmt.Println("<<<<<<<<<<<<<<<<<< nil >>>>>>>>>>>>>>>>>>")
		return
	}
	for len(statements) > 0 {
		levelLength := len(statements)
		fmt.Print("<<<<<<<<<<<<<<<<<<")
		for i := 0; i < levelLength; i++ {
			// 从队列中取出当前节点
			current := statements[0]
			statements = statements[1:]
			// 打印当前节点的值
			fmt.Print(current.TokenLiteral(), ",")

			// 将当前节点的所有子节点加入队列
			//for _, child := range current. {
			//	queue = append(queue, child)
			//}
		}
		// 每打印完一层后换行
		fmt.Println(">>>>>>>>>>>>>>>>>>")
	}
}
