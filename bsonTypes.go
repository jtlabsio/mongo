package querybuilder

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var reWord = regexp.MustCompile(`\w+`)

func detectDateComparisonOperator(field string, values []string) bson.M {
	if len(values) == 0 {
		return nil
	}

	// if values is greater than 0, use an $in clause
	if len(values) > 1 {
		a := bson.A{}

		// add each string value to the bson.A
		for _, v := range values {
			dv, _ := time.Parse(time.RFC3339, v)
			a = append(a, dv)
		}

		// create a filter with the array of values...
		filter := bson.M{
			field: bson.D{primitive.E{
				Key:   "$in",
				Value: a,
			}},
		}

		// return
		return filter
	}

	value := values[0]
	var oper string

	// check if string value is long enough for a 2 char prefix
	if len(value) >= 3 {
		var uv string

		// lte
		if value[0:2] == "<=" {
			oper = "$lte"
			uv = value[2:]
		}

		// gte
		if value[0:2] == ">=" {
			oper = "$gte"
			uv = value[2:]
		}

		// ne
		if value[0:2] == "!=" {
			oper = "$ne"
			uv = value[2:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// check if string value is long enough for a single char prefix
	if len(value) >= 2 {
		var uv string

		// lt
		if value[0:1] == "<" {
			oper = "$lt"
			uv = value[1:]
		}

		// gt
		if value[0:1] == ">" {
			oper = "$gt"
			uv = value[1:]
		}

		// ne
		if value[0:1] == "-" {
			oper = "$ne"
			uv = value[1:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// parse the date value
	dv, _ := time.Parse(time.RFC3339, value)
	var f interface{}

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		f = bson.D{primitive.E{
			Key:   oper,
			Value: dv,
		}}
	} else {
		f = dv
	}

	// return the filter
	return bson.M{field: f}
}

func detectNumericComparisonOperator(field string, values []string, numericType string) bson.M {
	if len(values) == 0 {
		return nil
	}

	var bitSize int
	switch numericType {
	case "decimal":
		bitSize = 32
	case "double":
		bitSize = 64
	case "int":
		bitSize = 32
	case "long":
		bitSize = 64
	default:
		return nil
	}

	// handle when values is an array
	if len(values) > 1 {
		a := bson.A{}

		for _, value := range values {
			var pv interface{}
			if numericType == "decimal" || numericType == "double" {
				v, _ := strconv.ParseFloat(value, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = float32(v)
				}
			} else {
				v, _ := strconv.ParseInt(value, 0, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = int32(v)
				}
			}

			a = append(a, pv)
		}

		// return a filter with the array of values...
		return bson.M{
			field: bson.D{primitive.E{
				Key:   "$in",
				Value: a,
			}},
		}
	}

	var oper string
	value := values[0]

	// check if string value is long enough for a 2 char prefix
	if len(value) >= 3 {
		var uv string

		// lte
		if value[0:2] == "<=" {
			oper = "$lte"
			uv = value[2:]
		}

		// gte
		if value[0:2] == ">=" {
			oper = "$gte"
			uv = value[2:]
		}

		// ne
		if value[0:2] == "!=" {
			oper = "$ne"
			uv = value[2:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// check if string value is long enough for a single char prefix
	if len(value) >= 2 {
		var uv string

		// lt
		if value[0:1] == "<" {
			oper = "$lt"
			uv = value[1:]
		}

		// gt
		if value[0:1] == ">" {
			oper = "$gt"
			uv = value[1:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	// parse the numeric value appropriately
	var parsedValue interface{}
	if numericType == "decimal" || numericType == "double" {
		v, _ := strconv.ParseFloat(value, bitSize)
		parsedValue = v

		// retype 32 bit
		if bitSize == 32 {
			parsedValue = float32(v)
		}
	} else {
		v, _ := strconv.ParseInt(value, 0, bitSize)
		parsedValue = v

		// retype 32 bit
		if bitSize == 32 {
			parsedValue = int32(v)
		}
	}

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		var clause bson.D
		if numericType == "decimal" || numericType == "double" {
			clause = bson.D{primitive.E{
				Key:   oper,
				Value: parsedValue,
			}}
		} else {
			clause = bson.D{primitive.E{
				Key:   oper,
				Value: parsedValue,
			}}
		}

		// return with the specified operator
		return bson.M{field: clause}
	}

	// no operator... just the value
	return bson.M{field: parsedValue}
}

func detectStringComparisonOperator(field string, values []string, bsonType string) bson.M {
	if len(values) == 0 {
		return nil
	}

	// if bsonType is object, query should use an exists operator
	if bsonType == "object" {
		filter := bson.M{}

		for _, fn := range values {
			// check for "-" prefix on field name
			exists := true
			if len(fn) >= 2 && fn[0:1] == "-" {
				exists = false
				fn = fn[1:]
			}

			// check for "!=" prefix on field name
			// NOTE: this is a bit of an odd syntax, but support was simple
			// to build in
			if exists && len(fn) >= 3 && fn[0:2] == "!=" {
				exists = false
				fn = fn[2:]
			}

			fn = fmt.Sprintf("%s.%s", field, fn)
			filter[fn] = bson.D{primitive.E{
				Key:   "$exists",
				Value: exists,
			}}
		}

		return filter
	}

	// if values is greater than 0, use an $in clause
	if len(values) > 1 {
		a := bson.A{}

		// add each string value to the bson.A
		for _, v := range values {
			a = append(a, v)
		}

		// when type is an array, don't use $in operator
		if bsonType == "array" {
			return bson.M{field: a}
		}

		// create a filter with the array of values using an $in operator for strings...
		return bson.M{field: bson.D{primitive.E{
			Key:   "$in",
			Value: a,
		}}}
	}

	// single value
	value := values[0]

	// ensure we have a word/value to filter with
	if !reWord.MatchString(value) {
		return nil
	}

	bw := false
	c := false
	em := false
	ew := false
	ne := false

	// check for prefix/suffix on the value string
	if len(value) > 1 {
		bw = value[len(value)-1:] == "*"
		ew = value[0:1] == "*"
		c = bw && ew
		ne = value[0:1] == "-"

		// not equal...
		if ne {
			return bson.M{field: bson.D{primitive.E{
				Key:   "$ne",
				Value: value[1:],
			}}}
		}
	}

	// check for != or string in quotes
	if len(value) > 2 {
		ne = value[0:2] == "!="
		em = value[0:1] == "\"" &&
			value[len(value)-1:] == "\""
	}

	// not equal...
	if ne {
		return bson.M{field: bson.D{primitive.E{
			Key:   "$ne",
			Value: value[2:],
		}}}
	}

	// contains...
	if c {
		return bson.M{field: primitive.Regex{
			Pattern: value[1 : len(value)-1],
			Options: "i",
		}}
	}

	// begins with...
	if bw {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s", value[0:len(value)-1]),
			Options: "i",
		}}
	}

	// ends with...
	if ew {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("%s$", value[1:]),
			Options: "i",
		}}
	}

	// exact match...
	if em {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s$", value[1:len(value)-1]),
			Options: "",
		}}
	}

	// the string value as is...
	return bson.M{field: value}
}

func combine(a bson.M, b bson.M) bson.M {
	for k, v := range b {
		a[k] = v
	}

	return a
}
