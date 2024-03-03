package repl

import (
	"bufio"
	"fmt"
	"io"
	"jonathan/lexer"
	"jonathan/parser"
)

const PROMPT = ">>"

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Printf(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.NewLexer(line)
		p := parser.NewParser(l)

		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}
		writeString, err := io.WriteString(out, program.String())
		if err != nil {
			fmt.Println("error writing string:", err)
			return
		}
		if writeString != len(program.String()) {
			fmt.Println("not all bytes written")
			return
		}

		writeString, err = io.WriteString(out, "\n")
		if err != nil {
			fmt.Println("error writing string:", err)
			return
		}
		if writeString != len("\n") {
			fmt.Println("not all bytes written")
			return
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	writeString, err := io.WriteString(out, " parser errors:\n")
	if err != nil {
		fmt.Println("error writing string:", err)
		return
	}
	if writeString != len("\n") {
		fmt.Println("not all bytes written")
		return
	}
	for _, msg := range errors {
		writeString, err = io.WriteString(out, "\t"+msg+"\n")
		if err != nil {
			fmt.Println("error writing string:", err)
			return
		}
		if writeString != len("\n") {
			fmt.Println("not all bytes written")
			return
		}
	}
}
