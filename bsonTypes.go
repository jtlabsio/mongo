package mongo

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var reWord = regexp.MustCompile(`\w+`)

func detectDateComparisonOperator(field string, values []string) bson.D {
	if len(values) == 0 {
		return nil
	}

	filter := bson.D{}

	for _, value := range values {
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

		// parse the date value
		dv, _ := time.Parse(time.RFC3339, value)
		f := primitive.E{
			Key: field,
		}

		// check if there is an lt, lte, gt or gte key
		if oper != "" {
			f.Value = bson.D{primitive.E{
				Key:   oper,
				Value: dv,
			}}
		} else {
			f.Value = dv
		}

		// add to the filter
		filter = append(filter, f)
	}

	return filter
}

func detectNumericComparisonOperator(field string, values []string, numericType string) bson.D {
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

	filter := bson.D{}

	for _, value := range values {
		clause := primitive.E{
			Key: field,
		}

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
			if numericType == "decimal" || numericType == "double" {
				clause.Value = bson.D{primitive.E{
					Key:   oper,
					Value: parsedValue,
				}}
			} else {
				clause.Value = bson.D{primitive.E{
					Key:   oper,
					Value: parsedValue,
				}}
			}

			// add to the clauses slice
			filter = append(filter, clause)
			continue
		}

		// no operator... just the value
		clause.Value = parsedValue

		// add to the clauses slice
		filter = append(filter, clause)
	}

	return filter
}

func detectStringComparisonOperator(field string, values []string, bsonType string) bson.D {
	if len(values) == 0 {
		return nil
	}

	// if bsonType is object, query should use an exists operator
	if bsonType == "object" {
		filter := bson.D{}

		for _, fn := range values {
			// check for "-" prefix on field name
			exists := true
			if len(fn) >= 2 && (fn[0:1] == "-" || fn[0:1] == "!") {
				exists = false
				fn = fn[1:]
			}

			fn = fmt.Sprintf("%s.%s", field, fn)
			filter = append(filter, primitive.E{
				Key: fn,
				Value: primitive.E{
					Key:   "$exists",
					Value: exists,
				},
			})
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

		// create a filter with the array of values...
		filter := bson.D{primitive.E{
			Key: field,
			Value: primitive.E{
				Key:   "$in",
				Value: a,
			},
		}}

		// return
		return filter
	}

	// single value
	value := values[0]

	// ensure we have a word/value to filter with
	if !reWord.MatchString(value) {
		return nil
	}

	bw := false
	c := false
	ew := false
	ne := false

	// check for prefix/suffix on the value string
	if len(value) > 2 {
		bw = value[len(value)-1:] == "*"
		ew = value[0:1] == "*"
		c = bw && ew
		ne = value[0:1] == "!"
	}

	// not exists...
	if ne {
		return bson.D{primitive.E{
			Key: field,
			Value: primitive.E{
				Key:   "$ne",
				Value: value[1:],
			},
		}}
	}

	// contains...
	if c {
		return bson.D{primitive.E{
			Key: field,
			Value: primitive.Regex{
				Pattern: value[1 : len(value)-1],
				Options: "i",
			},
		}}
	}

	// begins with...
	if bw {
		return bson.D{primitive.E{
			Key: field,
			Value: primitive.Regex{
				Pattern: fmt.Sprintf("^%s", value[0:len(value)-1]),
				Options: "i",
			},
		}}
	}

	// ends with...
	if ew {
		return bson.D{primitive.E{
			Key: field,
			Value: primitive.Regex{
				Pattern: fmt.Sprintf("%s$", value[1:]),
				Options: "i",
			},
		}}
	}

	// the string value as is...
	return bson.D{primitive.E{
		Key:   field,
		Value: value,
	}}
}

func combine(a bson.D, b bson.D) bson.D {
	a = append(a, b...)

	return a
}
