package querybuilder

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	reNull = regexp.MustCompile(`null`)
	reWord = regexp.MustCompile(`\w+`)
)

func iterateProperties(parentPrefix string, properties bson.M, ft *map[string]string) {
	// iterate each field within properties
	for field, value := range properties {
		switch value := value.(type) {
		case bson.M:
			// retrieve the type of the field
			if bsonType, ok := value["bsonType"]; ok {
				bsonType := bsonType.(string)
				// capture type in the fieldTypes map
				if bsonType != "" {
					(*ft)[fmt.Sprintf("%s%s", parentPrefix, field)] = bsonType
				}

				if bsonType == "array" {
					// look at "items" to get the bsonType
					if items, ok := value["items"]; ok {
						value = items.(bson.M)

						// fix for issue where Array of type strings is not properly
						// allowing filter with $in keyword
						if bsonType, ok := value["bsonType"]; ok {
							bsonType := bsonType.(string)
							// capture type in the fieldTypes map
							if bsonType != "" {
								(*ft)[fmt.Sprintf("%s%s", parentPrefix, field)] = bsonType
							}
						}
					}
				}

				// handle any sub-document schema details
				if subProperties, ok := value["properties"]; ok {
					subProperties := subProperties.(bson.M)
					iterateProperties(
						fmt.Sprintf("%s%s.", parentPrefix, field), subProperties, ft)
				}

				continue
			}

			// check for enum (without bsonType specified)
			if _, ok := value["enum"]; ok {
				(*ft)[fmt.Sprintf("%s%s", parentPrefix, field)] = "object"
			}
		default:
			// properties are not of type bson.M
			continue
		}
	}
}

func parseMapSchema(schema map[string]interface{}) map[string]string {
	// convert a map to a bson.M
	var conv func(map[string]any) bson.M
	conv = func(m map[string]any) bson.M {
		bm := bson.M{}
		for k, v := range m {
			if k == "bsonType" && reflect.TypeOf(v).String() == "[]string" {
				bm[k] = v.([]string)[0]
				continue
			}

			if sm, ok := v.(map[string]any); ok {
				bm[k] = conv(sm)
				continue
			}

			bm[k] = v
		}

		return bm
	}

	return parseBSONSchema(conv(schema))
}

func parseBSONSchema(schema bson.M) map[string]string {
	// check to see if top level is $jsonSchema
	if js, ok := schema["$jsonSchema"]; ok {
		schema = js.(bson.M)
	}

	// bsonType, required, properties at top level
	// looking for properties field, specifically
	flds := map[string]string{}
	if properties, ok := schema["properties"]; ok {
		properties := properties.(bson.M)
		iterateProperties("", properties, &flds)
	}

	// return empty map
	return flds
}

func parseJSONSchema(schema []byte) map[string]string {
	// convert JSON to a map
	m := map[string]any{}
	_ = bson.UnmarshalExtJSON(schema, false, &m)

	return parseMapSchema(m)
}

func parseStringSchema(schema string) map[string]string {
	// convert JSON string to a map
	m := map[string]any{}
	_ = bson.UnmarshalExtJSON([]byte(schema), false, &m)

	return parseMapSchema(m)
}

func parseUTCDate(value string) time.Time {
	dv, err := time.Parse(time.RFC3339, value)
	if err != nil {
		dv, err = time.Parse("2006-01-02", value)
		if err != nil {
			dv, _ = time.Parse("2006/01/02", value)
		}
	}

	return dv.UTC()
}

func detectComparisonOperator(value string, isTime bool) (string, string) {
	oper := ""
	if len(value) < 2 {
		return value, oper
	}

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

		if isTime && value[0:1] == "-" {
			oper = "$ne"
			uv = value[1:]
		}

		// update value to remove the prefix
		if uv != "" {
			value = uv
		}
	}

	return value, oper
}

func detectDateComparisonOperator(field string, values []string, lo LogicalOperator) bson.M {
	if len(values) == 0 {
		return nil
	}

	// if values is greater than 0, use an $in clause
	if len(values) > 1 {
		a := bson.A{}
		ina := bson.A{}
		op := false

		// add each string value to the bson.A
		for _, v := range values {
			v, oper := detectComparisonOperator(v, false)
			dv := parseUTCDate(v)

			// if there is an operator, structure the clause to include
			// the operator
			if oper != "" {
				op = true
				a = append(a, bson.D{bson.E{
					Key: field,
					Value: bson.D{bson.E{
						Key:   oper,
						Value: dv,
					}}}})
				continue
			}

			ina = append(ina, dv)
		}

		// determine type of query
		if op {
			// add any $in elements to the outer clause
			if len(ina) > 0 {
				a = append(a, bson.D{bson.E{
					Key: field,
					Value: bson.D{bson.E{
						Key:   "$in",
						Value: ina,
					}},
				}})
			}

			return bson.M{
				lo.String(): a,
			}
		}

		// return a filter with the array of values...
		return bson.M{
			field: bson.D{bson.E{
				Key:   "$in",
				Value: ina,
			}},
		}
	}

	// check for an operator in the value
	value, oper := detectComparisonOperator(values[0], true)

	// detect usage of keyword "null"
	if reNull.MatchString(value) {
		// check if there is an lt, lte, gt or gte key
		if oper != "" {
			return bson.M{field: bson.D{bson.E{
				Key:   oper,
				Value: nil,
			}}}
		}

		// return the filter
		return bson.M{field: nil}
	}

	// parse the date value
	dv := parseUTCDate(value)

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		return bson.M{field: bson.D{bson.E{
			Key:   oper,
			Value: dv,
		}}}
	}

	// return the filter
	return bson.M{field: dv}
}

func detectNumericComparisonOperator(field string, values []string, numericType string, lo LogicalOperator) bson.M {
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
		ina := bson.A{}
		op := false

		for _, value := range values {
			value, oper := detectComparisonOperator(value, false)

			var pv interface{}
			if numericType == "decimal" || numericType == "double" {
				v, _ := strconv.ParseFloat(value, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = float32(v)
				}
			}

			if pv == nil {
				v, _ := strconv.ParseInt(value, 0, bitSize)
				pv = v

				// retype 32 bit
				if bitSize == 32 {
					pv = int32(v)
				}
			}

			// if there is an operator, structure the clause to include
			// the operator
			if oper != "" {
				op = true
				a = append(a, bson.D{bson.E{
					Key: field,
					Value: bson.D{bson.E{
						Key:   oper,
						Value: pv,
					}}}})
				continue
			}

			// otherwise, just add the item to the list
			// TODO: lots of testing required here...
			// may need to add an operator when one isn't present
			// because the mongo query may be incorrect otherwise
			ina = append(ina, pv)
		}

		// determine type of query
		if op {
			// add any $in elements to the outer clause
			if len(ina) > 0 {
				a = append(a, bson.D{bson.E{
					Key: field,
					Value: bson.D{bson.E{
						Key:   "$in",
						Value: ina,
					}},
				}})
			}

			return bson.M{
				lo.String(): a,
			}
		}

		// return a filter with the array of values...
		return bson.M{
			field: bson.D{bson.E{
				Key:   "$in",
				Value: ina,
			}},
		}
	}

	// check for an operator in the value
	value, oper := detectComparisonOperator(values[0], false)

	if reNull.MatchString(value) {
		// aditionally detect $ne operator (note use of - shorthand here which
		// is not processed on numeric values that are not "null")... note: this
		// is detected here and not in the detectComparisonOperator because
		// numeric values with a prefix of "-" have meaning that is not the same
		// as an $ne comparison operator
		if value[0:1] == "-" {
			oper = "$ne"
		}

		if oper != "" {
			// return with the specified operator
			return bson.M{field: bson.D{bson.E{
				Key:   oper,
				Value: nil,
			}}}
		}

		return bson.M{field: nil}
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
	}

	if parsedValue == nil {
		v, _ := strconv.ParseInt(value, 0, bitSize)
		parsedValue = v

		// retype 32 bit
		if bitSize == 32 {
			parsedValue = int32(v)
		}
	}

	// check if there is an lt, lte, gt or gte key
	if oper != "" {
		// return with the specified operator
		return bson.M{field: bson.D{bson.E{
			Key:   oper,
			Value: parsedValue,
		}}}
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
			filter[fn] = bson.D{bson.E{
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
		return bson.M{field: bson.D{bson.E{
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

		// adjust value when not equal...
		if ne || ew {
			value = value[1:]
		}

		if bw {
			value = value[0 : len(value)-1]
		}

		if c {
			bw = false
			ew = false
		}
	}

	// check for != or string in quotes
	if len(value) > 2 && !ne {
		ne = value[0:2] == "!="
		em = value[0:1] == "\"" &&
			value[len(value)-1:] == "\""

		if ne {
			value = value[2:]
		}

		if em {
			value = value[1 : len(value)-1]
		}
	}

	// handle null keyword
	if reNull.MatchString(value) {
		if ne {
			return bson.M{field: bson.D{bson.E{
				Key:   "$ne",
				Value: nil,
			}}}
		}

		return bson.M{field: nil}
	}

	// not equal...
	if ne {
		return bson.M{field: bson.D{bson.E{
			Key:   "$ne",
			Value: value,
		}}}
	}

	// contains...
	if c {
		return bson.M{field: primitive.Regex{
			Pattern: value,
			Options: "i",
		}}
	}

	// begins with...
	if bw {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s", value),
			Options: "i",
		}}
	}

	// ends with...
	if ew {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("%s$", value),
			Options: "i",
		}}
	}

	// exact match...
	if em {
		return bson.M{field: primitive.Regex{
			Pattern: fmt.Sprintf("^%s$", value),
			Options: "",
		}}
	}

	// the string value as is...
	return bson.M{field: value}
}

func combine(a bson.M, b bson.M) bson.M {
	for k, v := range b {
		// check for existing key
		if ev, ok := a[k]; ok {
			// check if existing value is a bson.M
			if evm, ok := ev.(bson.M); ok {
				// check if new value is a bson.M
				if vm, ok := v.(bson.M); ok {
					// combine the two bson.M values
					a[k] = combine(evm, vm)
					continue
				}
			}

			// check if existing value is a bson.A
			if eva, ok := ev.(bson.A); ok {
				// check if new value is a bson.A
				if va, ok := v.(bson.A); ok {
					// combine the two bson.A values by appending each element
					// from the second array to the first
					a[k] = append(eva, va...)
					continue
				}
			}
		}

		a[k] = v
	}

	return a
}
