package parser

import (
	"jonathan/ast"
	"jonathan/lexer"
	"jonathan/token"
)

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}

	// read two token  set curToken and peekToken  why? TODO
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	//program = newProgramASTNode()
	//
	//advanceToken()
	//
	//for currentToken() != EOF_TOKEN {
	//	statement = null
	//
	//	if currentToken() == LET_TOKEN {
	//		statment = parseLetStatement()
	//	} else if currentToken() == IF_TOKEN {
	//		statement = parseIfStatement()
	//	}
	//
	//	if statment {
	//
	//	}
	//}

	return nil
}
