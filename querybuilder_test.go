package mongo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func Test_NewQueryBuilder(t *testing.T) {
	type args struct {
		collection string
		schema     bson.M
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test with basic schema",
			args: args{
				collection: "test",
				schema: bson.M{
					"bsonType": "object",
					"required": []string{"someID", "created", "someName"},
					"properties": bson.M{
						"someID": bson.M{
							"bsonType":    "string",
							"description": "primary identifier of something, must be unique",
						},
						"created": bson.M{
							"bsonType":    "date",
							"description": "date for when the thing was created",
						},
						"someName": bson.M{
							"bsonType":    "string",
							"description": "string name of the thing",
						},
						"disabled": bson.M{
							"bsonType":    "bool",
							"description": "boolean type",
						},
						"customEnum": bson.M{
							"enum":        bson.A{"A", "B", "C"},
							"description": "an enum type",
						},
						"minMaxNumber": bson.M{
							"bsonType":    "int",
							"minimum":     0,
							"maximum":     100,
							"description": "number with a min and max",
						},
						"childStructureNoSchema": bson.M{
							"bsonType":    "object",
							"description": "child structure with no schema",
						},
						"childStructure": bson.M{
							"bsonType": "object",
							"required": bson.A{},
							"properties": bson.M{
								"fieldA": bson.M{
									"bsonType":    "array",
									"description": "an array of elements",
								},
								"fieldB": bson.M{
									"bsonType":    "date",
									"description": "a nested date value",
								},
								"fieldC": bson.M{
									"bsonType": "object",
									"required": bson.A{"fieldC-1"},
									"properties": bson.M{
										"fieldC-1": bson.M{
											"bsonType":    "string",
											"description": "nested two layers deep string",
										},
										"fieldC-2": bson.M{
											"bsonType":    "double",
											"description": "a double value",
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[string]string{
				"someID":                         "string",
				"created":                        "date",
				"someName":                       "string",
				"disabled":                       "bool",
				"minMaxNumber":                   "int",
				"childStructureNoSchema":         "object",
				"childStructure":                 "object",
				"childStructure.fieldB":          "date",
				"childStructure.fieldC":          "object",
				"childStructure.fieldC.fieldC-1": "string",
				"childStructure.fieldC.fieldC-2": "double",
				"childStructure.fieldA":          "array",
				"customEnum":                     "object",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(tt.args.collection, tt.args.schema)
			if !reflect.DeepEqual(qb.fieldTypes, tt.want) {
				t.Errorf("NewQueryBuilder(), qb.fieldTypes = %v, want %v", qb.fieldTypes, tt.want)
			}
		})
	}
}
