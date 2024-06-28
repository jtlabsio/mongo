package querybuilder

import (
	"reflect"
	"strings"
)

func forEachField(val reflect.Value, pfx string, call func(string, any) error) error {
	// if the doc is a pointer, dereference it
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// ensure the doc is a struct (nothing to build an update for otherwise)
	if val.Kind() != reflect.Struct {
		return nil
	}

	// iterate over each field in the struct
	for i := 0; i < val.NumField(); i++ {
		fldV := val.Field(i)
		fldF := val.Type().Field(i)

		// determine if the field is exported
		if fldF.PkgPath != "" {
			continue
		}

		nm := getMongoFieldName(fldF)
		if pfx != "" {
			nm = strings.Join([]string{pfx, nm}, ".")
		}

		// if the field is a struct, recurse into it
		if fldV.Kind() == reflect.Struct || fldV.Kind() == reflect.Ptr && fldV.Elem().Kind() == reflect.Struct {
			isTime := fldV.Type().String() == "time.Time"
			if fldV.Kind() == reflect.Ptr && fldV.Elem().Type().String() == "time.Time" {
				isTime = true
			}

			// make sure field is not a Time
			if !isTime {
				if err := forEachField(fldV, nm, call); err != nil {
					return err
				}

				continue
			}
		}

		// callback with the dot notation path and the value
		if err := call(nm, getValue(fldV)); err != nil {
			return err
		}
	}

	return nil
}

func getMongoFieldName(fld reflect.StructField) string {
	// pull the bson tag from the field
	tag := fld.Tag.Get("bson")

	// fallback to json tag if bson tag is empty
	if tag == "" {
		tag = fld.Tag.Get("json")
	}

	// strip out any omitempty or other flags
	tag = strings.Split(tag, ",")[0]

	return tag
}

func getValue(val reflect.Value) any {
	return val.Interface()
}

func isValueEmpty(val any) bool {
	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return true
	}

	// if the value is a pointer, slice, map, function, or interface
	// look for nil value
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func, reflect.Interface:
		return v.IsNil()
	}

	// if the value implements IsZero, return the result of that
	type zero interface {
		IsZero() bool
	}
	if z, ok := val.(zero); ok {
		return z.IsZero()
	}

	// if value is a string, look for blank string
	if v.Kind() == reflect.String {
		return v.String() == ""
	}

	// if value is numeric, look for zero value
	if v.Kind() >= reflect.Int && v.Kind() <= reflect.Float64 {
		return v.Interface() == reflect.Zero(v.Type()).Interface()
	}

	// if we got this far, there must be a value
	return false
}
