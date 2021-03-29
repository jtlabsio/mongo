package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

type QueryBuilder struct {
	fieldTypes map[string]string
}

func NewQueryBuilder(schema bson.M) *QueryBuilder {
	qb := QueryBuilder{
		fieldTypes: map[string]string{},
	}

	if schema != nil {
		qb.discoverFields(schema)
	}

	return &qb
}

func (qb QueryBuilder) discoverFields(schema bson.M) {
	// ensure fieldTypes is set
	if qb.fieldTypes == nil {
		qb.fieldTypes = map[string]string{}
	}

	// bsonType, required, properties at top level
	// looking for properties field, specifically
	if properties, ok := schema["properties"]; ok {
		properties := properties.(bson.M)
		qb.iterateProperties("", properties)
	}
}

func (qb QueryBuilder) iterateProperties(parentPrefix string, properties bson.M) {
	// iterate each field within properties
	for field, value := range properties {
		switch value := value.(type) {
		case bson.M:
			// retrieve the type of the field
			if bsonType, ok := value["bsonType"]; ok {
				bsonType := bsonType.(string)
				// capture type in the fieldTypes map
				if bsonType != "" {
					qb.fieldTypes[fmt.Sprintf("%s%s", parentPrefix, field)] = bsonType
				}

				// handle any sub-document schema details
				if subProperties, ok := value["properties"]; ok {
					subProperties := subProperties.(bson.M)
					qb.iterateProperties(
						fmt.Sprintf("%s%s.", parentPrefix, field), subProperties)
				}

				continue
			}

			// check for enum (without bsonType specified)
			if _, ok := value["enum"]; ok {
				qb.fieldTypes[fmt.Sprintf("%s%s", parentPrefix, field)] = "object"
			}
		default:
			// unknown type
			continue
		}
	}
}
