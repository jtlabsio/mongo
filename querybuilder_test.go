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
							"description": "name of the thing",
						},
						"disabled": bson.M{
							"bsonType":    "",
							"description": "",
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
