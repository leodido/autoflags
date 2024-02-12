package autoflags

import (
	"reflect"
	"strconv"
)

func isMandatory(f reflect.StructField) bool {
	val := f.Tag.Get("flagrequired")
	req, _ := strconv.ParseBool(val)

	return req
}
