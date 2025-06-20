package internaltag

import (
	"reflect"
	"regexp"
	"strconv"
)

var validFlagNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+([.-][a-zA-Z0-9]+)*$`)

func IsValidFlagName(name string) bool {
	return validFlagNameRegex.MatchString(name)
}

var standardTypes = func() map[reflect.Kind]reflect.Type {
	types := make(map[reflect.Kind]reflect.Type)
	for _, v := range []any{
		"", int(0), bool(false), int8(0), int16(0), int32(0), int64(0),
		uint(0), uint8(0), uint16(0), uint32(0), uint64(0),
	} {
		t := reflect.TypeOf(v)
		types[t.Kind()] = t
	}
	return types
}()

func IsStandardType(t reflect.Type) bool {
	expected, exists := standardTypes[t.Kind()]

	return exists && t == expected
}

func IsMandatory(f reflect.StructField) bool {
	val := f.Tag.Get("flagrequired")
	req, _ := strconv.ParseBool(val)

	return req
}
