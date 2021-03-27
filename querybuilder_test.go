package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestQueryBuilder_discoverFields(t *testing.T) {
	type args struct {
		schema bson.M
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test with basic schema",
			args: args{
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
						"childStructure": bson.M{
							"bsonType": "object",
							"required": bson.A{},
							"properties": bson.M{
								"fieldA": bson.M{
									"bsonType":    "",
									"description": "",
								},
								"fieldB": bson.M{
									"bsonType":    "",
									"description": "",
								},
								"fieldC": bson.M{
									"bsonType":   "object",
									"required":   bson.A{},
									"properties": bson.M{},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := QueryBuilder{}
			if err := qb.discoverFields(tt.args.schema); (err != nil) != tt.wantErr {
				t.Errorf("QueryBuilder.discoverFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
