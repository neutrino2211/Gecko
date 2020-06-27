package utils

import (
	"os"
	"strings"

	"github.com/neutrino2211/Gecko/ast"
	"github.com/neutrino2211/Gecko/tokens"
)

func IsBool(b interface{}) bool {

	switch b.(type) {
	case bool:
		return true
	}

	return false
}

func IsFuncCall(c interface{}) bool {
	switch c.(type) {
	case *tokens.FuncCall:
		return true
	}

	return false
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ResolveVariable(scope *ast.Ast, symbol string) *ast.Variable {
	variable := scope.Variables[symbol]
	if strings.Contains(symbol, ".") {
		vars := strings.Split(symbol, ".")
		variable = scope.Variables[vars[0]]
	} else if variable == nil && scope.Parent != nil && scope.Parent.Methods[scope.Name] != nil {
		vScope := scope.Parent.Methods[scope.Name].ToAst()
		variable = vScope.Variables[symbol]
	}

	return variable
}
