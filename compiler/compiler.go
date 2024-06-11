package compiler

import (
	"jonathan/ast"
	"jonathan/code"
	"jonathan/object"
)

type Compiler struct {
	instructions code.Instructions
	constants    []object.Object //constant pool
}

func NewCompiler() *Compiler {
	return &Compiler{
		instructions: code.Instructions{},
		constants:    []object.Object{},
	}
}
func (c *Compiler) Compile(node ast.Node) error {
	return nil
}
func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}
