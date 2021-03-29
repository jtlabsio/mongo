package mongo

import (
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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

		// lte
		if value[0:2] == "<=" {
			oper = "$lte"
			value = value[2:]
		}

		// gte
		if value[0:2] == ">=" {
			oper = "$gte"
			value = value[2:]
		}

		// lt
		if value[0:1] == "<" {
			oper = "$lt"
			value = value[1:]
		}

		// gt
		if value[0:1] == ">" {
			oper = "$gt"
			value = value[1:]
		}

		var parsedValue interface{}
		if numericType == "decimal" || numericType == "double" {
			v, _ := strconv.ParseFloat(value, bitSize)
			parsedValue = v
		} else {
			v, _ := strconv.ParseInt(value, 0, bitSize)
			parsedValue = v
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

func combine(a bson.D, b bson.D) bson.D {
	for _, e := range b {
		a = append(a, e)
	}

	return a
}
