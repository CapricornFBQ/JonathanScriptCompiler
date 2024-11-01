package parser

import (
	"fmt"
	"jonathan/ast"
	"jonathan/lexer"
	"jonathan/token"
	"log"
	"reflect"
	"strconv"
)

// level============================  level

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         //+
	PRODUCT     //*
	PREFIX      //-Xor!X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[token.Type]int{
	token.EQ:       EQUALS,
	token.NotEq:    EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l                       *lexer.Lexer
	errors                  []string
	curToken                token.Token
	peekToken               token.Token
	currentParsedStatements []ast.Statement // used in printer

	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn
}

func NewParser(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// prefix parse
	p.prefixParseFns = make(map[token.Type]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)

	// infix parse
	p.infixParseFns = make(map[token.Type]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NotEq, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	// read two token  set curToken and peekToken  why? TODO
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// ParseProgram start parse program===============================================  start parse program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}
	// the biggest loop. if parse statement return nil , the loop continue.
	// the statements is a slice ,include all the statements parsed in the program
	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement() // the ast.Statement is implemented in pointer type ,so we have to assign it with pointer!
		if pointerValue, ok := stmt.(ast.Statement); ok {
			// use the reflect to check nil
			if !reflect.ValueOf(pointerValue).IsNil() {
				program.Statements = append(program.Statements, pointerValue)
				p.currentParsedStatements = append(p.currentParsedStatements, pointerValue)
			}
		}
		p.nextToken()
	}
	return program
}

// parseStatement and specific logic===============================================  parseStatement and specific logic
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// Check whether the following token meets the requirements of the statement
func (p *Parser) parseLetStatement() *ast.LetStatement {
	defer unTrace(trace("parseLetStatement", p))
	stmt := &ast.LetStatement{Token: p.curToken}
	if !p.expectPeek(token.IDENT) {
		p.printNilInfo()
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	if !p.expectPeek(token.ASSIGN) {
		p.printNilInfo()
		return nil
	}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	if fl, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		fl.Name = stmt.Name.Value
	}
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	fmt.Printf("stmt: %+v\n", stmt)
	fmt.Printf("name: %+v\n", *stmt.Name)
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	defer unTrace(trace("parseReturnStatement", p))
	stmt := &ast.ReturnStatement{Token: p.curToken}
	p.nextToken()
	stmt.ReturnValue = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseExpression and specific logic===============================================  parseExpression and specific logic
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	defer unTrace(trace("parseExpressionStatement", p))
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// !!! Note: Prioritize high priority (deal with the next expression). [equal to：precedence < p.peekPrecedence() return true]
// If break , it will return left Exp, that mean
// "parseInfixExpression function - expression.Right" will get a result (deal with the current expression)[equal to：precedence < p.peekPrecedence() return false]
// parseExpression,parsePrefixExpression and parseInfixExpression constitutes complete recursion.
// The result of the parseInfixExpression function serves as the left child node in the subsequent infix expression tree.
// The result of the parseExpression function becomes the right child of the right subtree within a PrefixExpression in the AST.
// Character expressions：2*3+4+5-6*7
// expression node result：
//
//          -
//        /   \
//       +     *
//      / \   / \
//     +   5 6   7
//    / \
//   *   4
//  / \
// 2   3

// Character expressions：a + b * c + d / e - f
// expression node result：
//
//         -
//        / \
//       +   f
//      / \
//     +   /
//    / \ / \
//   a  * d  e
//     / \
//    b   c

func (p *Parser) parseExpression(precedence int) ast.Expression {
	defer unTrace(trace("parseExpression", p))
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		p.printNilInfo()
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		// curToken is the left node
		p.nextToken()
		// infix function return a expression node
		leftExp = infix(leftExp) // pass the left operand into the infix parse function then get the operator and right operand
		//The leftExp === A tree-like data structure is formed through recursion

	}

	return leftExp
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	defer unTrace(trace("parsePrefixExpression", p))
	// curToken should be an operator
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)
	return expression
}

// InfixExpression    ====================
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	defer unTrace(trace("parseInfixExpression", p))
	// curToken should be an infix operator
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)
	return expression
}

// Identifier         ====================
func (p *Parser) parseIdentifier() ast.Expression {
	defer unTrace(trace("parseIdentifier", p))
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// String             ====================
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// PrefixParseFnError ====================
func (p *Parser) noPrefixParseFnError(t token.Type) {
	defer unTrace(trace("noPrefixParseFnError", p))
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// IntegerLiteral     ====================
func (p *Parser) parseIntegerLiteral() ast.Expression {
	defer unTrace(trace("parseIntegerLiteral", p))
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		p.printNilInfo()
		return nil
	}
	lit.Value = value
	return lit
}

// Boolean           ====================
func (p *Parser) parseBoolean() ast.Expression {
	defer unTrace(trace("parseBoolean", p))
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// GroupedExpression ====================
func (p *Parser) parseGroupedExpression() ast.Expression {
	defer unTrace(trace("parseGroupedExpression", p))
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		p.printNilInfo()
		return nil
	}
	return exp
}

// BlockStatement  ====================
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	defer unTrace(trace("parseBlockStatement", p))
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	return block
}

// IfExpression     ====================
func (p *Parser) parseIfExpression() ast.Expression {
	defer unTrace(trace("parseIfExpression", p))
	expression := &ast.IfExpression{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		p.printNilInfo()
		return nil
	}
	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		p.printNilInfo()
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		p.printNilInfo()
		return nil
	}
	expression.Consequence = p.parseBlockStatement()
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) {
			p.printNilInfo()
			return nil
		}
		expression.Alternative = p.parseBlockStatement()
	}
	return expression
}

// FunctionParameters====================
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	defer unTrace(trace("parseFunctionParameters", p))
	var identifiers []*ast.Identifier
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}
	p.nextToken()
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}
	if !p.expectPeek(token.RPAREN) {
		p.printNilInfo()
		return nil
	}
	return identifiers
}

// FunctionLiteral   ====================
func (p *Parser) parseFunctionLiteral() ast.Expression {
	defer unTrace(trace("parseFunctionLiteral", p))
	lit := &ast.FunctionLiteral{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		p.printNilInfo()
		return nil
	}
	lit.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		p.printNilInfo()
		return nil
	}
	lit.Body = p.parseBlockStatement()
	return lit
}

// CallExpression    ====================
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	defer unTrace(trace("parseCallExpression", p))
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN) // arguments parse is the same to the array list
	return exp
}

// ArrayLiteral    ====================
func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

// ArrayLiteral  split with ","  ====================
func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	var list []ast.Expression
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

// IndexExpression  ==========================
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return exp
}

// HashLiteral  ==========================
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)
	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		value := p.parseExpression(LOWEST)
		hash.Pairs[key] = value
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return hash
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// helper func ============================================================================================ helper fun//
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekPrecedence() int {
	if pre, ok := precedences[p.peekToken.Type]; ok {
		return pre
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pre, ok := precedences[p.curToken.Type]; ok {
		return pre
	}
	return LOWEST
}

func (p *Parser) peekError(t token.Type) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefix(tokenType token.Type, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.Type, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) printNilInfo() {
	log.Printf("[[[[[[[[[[[[[[current literal: [ %s ] , next literal: [ %s ]]]]]]]]]]]]]]]", p.curToken.Literal, p.peekToken.Literal)
}
