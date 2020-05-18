package utils

import (
	"strings"
)

func ParseString(v string) string {
	return strings.ReplaceAll(v[1:len(v)-1], "\\", "")
}
