package errors

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/alecthomas/participle/lexer"
	"github.com/neutrino2211/Gecko/ast"
)

type Error struct {
	Pos    lexer.Position
	Reason string
	Scope  *ast.Ast
}

func computeStackTrace(scope *ast.Ast) string {
	var s = ""
	currScope := scope
	for currScope.Parent != nil {
		s += "\t-> " + currScope.Parent.Name + "." + currScope.Name + "\n"
		currScope = currScope.Parent
	}
	s += "\t-> " + currScope.Name
	return s
}

func (e *Error) getErrorLine() string {
	byts, err := ioutil.ReadFile(e.Pos.Filename)
	if err == nil {
		lines := strings.Split(string(byts), "\n")
		return lines[e.Pos.Line-1]
	}

	return ""
}

func (e *Error) String() string {
	return fmt.Sprintf("Error: %s [%s]\n\t%s\n\n%s\n", e.Reason, e.Pos.String(), e.getErrorLine(), computeStackTrace(e.Scope))
}

func (e *Error) Error() string {
	return e.String()
}

func NewError(pos lexer.Position, reason string, scope *ast.Ast) *Error {
	return &Error{
		Pos:    pos,
		Reason: reason,
		Scope:  scope,
	}
}

var errors = []*Error{}
var ignoreNext = false
var errorWasIgnored = false

func AddError(err *Error) {
	println(ignoreNext, errorWasIgnored)
	if ignoreNext && !errorWasIgnored {
		errorWasIgnored = true
		return
	}
	errorWasIgnored = false
	ignoreNext = false
	errors = append(errors, err)
}

func GetErrors() []*Error {
	return errors
}

func IgnoreNextError() {
	ignoreNext = true
	// println("Error:::", ignoreNext)
}

func ErrorWasIgnored() bool {
	return errorWasIgnored
}

func HaveErrors() bool {
	return len(errors) != 0
}
