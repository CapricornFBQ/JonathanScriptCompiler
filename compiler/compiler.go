package compiler

import (
	"fmt"
	"jonathan/ast"
	"jonathan/code"
	"jonathan/object"
	"log"
	"sort"
)

// EmittedInstruction for jump
type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

// static compile scope, used for controlling the instructions generation
// it's the item in a two-dimensional instructions arrays, every item is a instructions array. every item is independent of each other.
type CompilationScope struct {
	instructions        code.Instructions  // the result instructions byte[] the compiler generated
	lastInstruction     EmittedInstruction // the last emitted instruction
	previousInstruction EmittedInstruction // the one before last emitted instruction
}

type Compiler struct {
	constants   []object.Object // constant pool
	symbolTable *SymbolTable    // trace the scope of symbol
	// it's convenient control of scopes when we decode the instruction
	scopes     []CompilationScope // trace the instruction emitted. the instructions is a two-dimensional instructions arrays
	scopeIndex int                // the index of scope depth
}

func NewCompiler() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	symbolTable := NewSymbolTable()
	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}
	return &Compiler{
		constants:   []object.Object{},
		symbolTable: symbolTable,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func NewCompilerWithState(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := NewCompiler()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)
	case *ast.Identifier:
		// this will be executed, when the expression operand is a identifier
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value)
		}
		c.loadSymbol(symbol)
	case *ast.LetStatement:
		// Note: compile expression first. if the value is a integer. the previous instruction is a integer.
		// the integer will be pushed into the stack in the vm. so, when the vm get the OpSetGlobal ,
		// it will get the integer by pop the stack. and store the integer in the array at the index of symbol.index.
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		symbol := c.symbolTable.Define(node.Name.Value)
		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}
	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}
		// Emit an `OpJumpNotTruthy` with a bogus value. and get the position of `OpJumpNotTruthy`
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)
		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}
		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}
		// Emit an `OpJump` with a bogus value
		jumpPos := c.emit(code.OpJump, 9999)
		// the position that 'no truthy' condition should jump to，the alternative expression
		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}
		// the position that 'truthy' condition should jump to，the position after alternative expression
		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}
		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	case *ast.InfixExpression:
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThan)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}
		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case ">":
			c.emit(code.OpGreaterThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))
	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpArray, len(node.Elements))
	case *ast.HashLiteral:
		var keys []ast.Expression
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
		// key and valve compiled into the instructions, then generate opHash instruction
		for _, k := range keys {
			err := c.Compile(k)
			if err != nil {
				return err
			}
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return err
			}
		}
		c.emit(code.OpHash, len(node.Pairs)*2)
	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(code.OpIndex)
	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	case *ast.FunctionLiteral:
		c.enterScope()
		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}
		err := c.Compile(node.Body)
		if err != nil {
			return err
		}
		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}
		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}
		freeSymbols := c.symbolTable.FreeSymbols
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope()
		for _, s := range freeSymbols { // emit free symbol before emit closure
			c.loadSymbol(s)
		}
		compiledFn := &object.CompiledFunction{
			Instructions:  instructions,
			NumLocals:     numLocals,
			NumParameters: len(node.Parameters),
		}
		fnIndex := c.addConstant(compiledFn)
		c.emit(code.OpClosure, fnIndex, len(freeSymbols))

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}
		c.emit(code.OpReturnValue)
	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}
		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpCall, len(node.Arguments))

	}
	c.PrintStatements()
	return nil
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))
	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction
	old := c.currentInstructions()
	new := old[:last.Position]
	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.currentInstructions(),
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

// Store the operand object and get its index, then store the index in the instruction
func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

// Get instruction of the op,and add it into instructions. return the start position of op instruction
func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

// add the new instruction into the instruction array
func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)

	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return posNewInstruction
}

// record the last instruction
func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

// replace the instruction on the position
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]

	}
}

// change the operand of the instruction and replace
func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	newInstruction := code.Make(op, operand)
	c.replaceInstruction(opPos, newInstruction)
}

// create a new instructions array
func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable) // new symbol table of scope
}

// remove the last instructions array
func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()
	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--
	c.symbolTable = c.symbolTable.Outer // come back to the outer symbol table of scope
	return instructions
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	default:
	}
}

func (c *Compiler) PrintStatements() {
	fmt.Println("------------compiler status:------------")
	log.Printf("\nconstants size: %d ", len(c.constants))
	fmt.Println("compiler constants:")
	for i := 0; i < len(c.constants); i++ {
		fmt.Printf("obj: %v ,", c.constants[i])
		// fmt.Printf("str: %s ,", c.constants[i])
		// fmt.Printf("hex: %x ,", c.constants[i])
	}
	log.Printf("\nconstants symbolTable: %d ", c.symbolTable.numDefinitions)
	fmt.Printf("\ncompiler symbolTable: ")
	for key, value := range c.symbolTable.store {
		fmt.Printf("\nkey:%s=>Name:%s, Scope: %s, Index: %d ", key, value.Name, value.Scope, value.Index)
	}
	fmt.Println("compiler scope: ")
	log.Printf("\ncurrent scope index: %d ", c.scopeIndex)
	for i := 0; i < len(c.scopes); i++ {
		fmt.Printf("compiler scope num: %d ", i)
		fmt.Printf("compiler instructions: %s;;;;", c.scopes[i].instructions)
	}
}
