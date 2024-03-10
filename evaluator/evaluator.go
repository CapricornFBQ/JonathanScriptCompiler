package evaluator

import (
	"jonathan/ast"
	"jonathan/object"
)

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {
	// Statements
	case *ast.Program:
		return evalStatements(node.Statements)
	case *ast.ExpressionStatement:
		return Eval(node.Expression)
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	}
	//  another implement
	//case *ast.ExpressionStatement:
	//	switch expression := node.Expression.(type) {
	//	case *ast.IntegerLiteral:
	//		return &object.Integer{Value: expression.Value}
	//	}
	//}
	return nil
}

func evalStatements(stmts []ast.Statement) object.Object {
	var result object.Object
	for _, statement := range stmts {
		result = Eval(statement)
	}
	return result
}
