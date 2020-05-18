package utils

import "reflect"

func TypeOf(i interface{}) reflect.Type {
	return reflect.TypeOf(i)
}
