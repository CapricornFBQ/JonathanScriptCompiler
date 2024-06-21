package repl

import (
	"bufio"
	"fmt"
	"io"
	"jonathan/compiler"
	"jonathan/lexer"
	"jonathan/parser"
	"jonathan/vm"
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

		comp := compiler.NewCompiler()
		err := comp.Compile(program)
		if err != nil {
			_, err := fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			if err != nil {
				return
			}
			continue
		}
		machine := vm.NewVm(comp.Bytecode())
		err = machine.Run()
		if err != nil {
			_, err := fmt.Fprintf(out, "Woops! Executing bytecode failed:\n %s\n", err)
			if err != nil {
				return
			}
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		_, err = io.WriteString(out, lastPopped.Inspect())
		if err != nil {
			return
		}
		_, err = io.WriteString(out, "\n")
		if err != nil {
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
