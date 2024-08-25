package compiler

type SymbolScope string

const (
	GlobalScope SymbolScope = "GLOBAL"
)

// trace the scope of symbol(variable or function literal especially)
type Symbol struct {
	Name  string      // symbol name
	Scope SymbolScope // symbol scope
	Index int         // symbol index
}
type SymbolTable struct {
	store          map[string]Symbol
	numDefinitions int
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	return &SymbolTable{store: s}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{Name: name, Index: s.numDefinitions, Scope: GlobalScope}
	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	return obj, ok
}
