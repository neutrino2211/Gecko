package utils

import (
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
