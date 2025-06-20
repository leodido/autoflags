package internalpath

import (
	"fmt"
	"reflect"
	"strings"
)

func GetName(name, alias string) string {
	res := name
	if alias != "" {
		res = alias
	}

	return res
}

func GetFieldName(prefix string, structField reflect.StructField) string {
	if prefix == "" {
		return structField.Name
	}
	return prefix + "." + structField.Name
}

func GetFieldPath(structPath string, structField reflect.StructField) string {
	path := ""
	if structPath == "" {
		path = strings.ToLower(structField.Name)
	} else {
		path = fmt.Sprintf("%s.%s", structPath, strings.ToLower(structField.Name))
	}
	return path
}
