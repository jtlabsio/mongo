package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

type QueryBuilder struct {
	fieldTypes map[string]string
}

func NewQueryBuilder(schema bson.M) *QueryBuilder {
	qb := QueryBuilder{}

	if schema != nil {
		qb.discoverFields(schema)
	}

	return &qb
}

func (qb QueryBuilder) discoverFields(schema bson.M) error {
	// bsonType, required, properties at top level
	// looking for properties field, specifically
	if properties, ok := schema["properties"]; ok {
		properties := properties.(bson.M)
		for key, value := range properties {
			value := value.(bson.M)
			if bsonType, ok := value["bsonType"]; ok {
				fmt.Println(key, bsonType)
			}
		}
	}

	return nil
}
