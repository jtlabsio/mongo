package querybuilder

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type thing struct {
	ThingID    string    `bson:"thingID"`
	Name       string    `bson:"name"`
	Created    time.Time `bson:"created"`
	Attributes []string  `bson:"attributes,omitempty"`
	SubThing   *subThing `bson:"sub,omitempty"`
}

type subThing struct {
	SubThingID string    `json:"subThingID"`
	Name       string    `json:"name"`
	Created    time.Time `json:"created"`
	Attributes []string  `json:"attributes,omitempty"`
}

var thingsSchema = `
{
	"$jsonSchema": {
		"bsonType": "object",
		"required": ["thingID"],
		"properties": {
			"thingID": {
				"bsonType":    "string",
				"description": "primary identifier for the thing"
			},
			"created": {
				"bsonType":    "date",
				"description": "time at which the thing was created"
			},
			"name": {
				"bsonType":    "string",
				"description": "name of the thing"
			},
			"attributes": {
				"bsonType":    "array",
				"description": "type tags for the thing",
				"items": {
					"bsonType": "string"
				}
			},
			"sub": {
				"bsonType": "object",
				"properties": {
					"subThingID": {
						"bsonType":    "string",
						"description": "primary identifier for the sub thing"
					},
					"created": {
						"bsonType":    "date",
						"description": "time at which the sub thing was created"
					},
					"name": {
						"bsonType":    "string",
						"description": "name of the sub thing"
					},
					"attributes": {
						"bsonType":    "array",
						"description": "type tags for the sub thing",
						"items": {
							"bsonType": "string"
						}
					}
				}
			}
		}
	}
}`

func TestUpdateBuilder_Update(t *testing.T) {
	type fields struct {
		collection       string
		fieldTypes       map[string]string
		strictValidation bool
	}
	type args struct {
		doc any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.D
		wantErr bool
	}{ /*
			{
				"should error if doc is not a struct",
				fields{},
				args{
					"testing",
				},
				bson.D{},
				true,
			},//*/
		{
			"should handle when doc is a pointer",
			fields{},
			args{
				&thing{
					ThingID: "123",
				},
			},
			bson.D{},
			true,
		},
		{
			"Test UpdateBuilder Update",
			fields{
				collection:       "things",
				fieldTypes:       parseSchema(thingsSchema),
				strictValidation: false,
			},
			args{
				thing{
					ThingID:    "123",
					Name:       "thing",
					Created:    time.Now(),
					Attributes: []string{"tag1", "tag2"},
					SubThing: &subThing{
						SubThingID: "456",
						Name:       "subthing",
						Created:    time.Now(),
						Attributes: []string{"tag3", "tag4"},
					},
				},
			},
			bson.D{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ub := &UpdateBuilder{
				collection:       tt.fields.collection,
				fieldTypes:       tt.fields.fieldTypes,
				strictValidation: tt.fields.strictValidation,
			}

			got, err := ub.Update(tt.args.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBuilder.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				gj, err := bson.MarshalExtJSONIndent(got, false, false, "", "  ")
				if err != nil {
					fmt.Printf("\n\n%+v\n\n", got)
					fmt.Printf("TEST ERROR: %v\n\n", err)
				}
				wj, _ := bson.MarshalExtJSONIndent(tt.want, false, false, "", "  ")
				t.Errorf("UpdateBuilder.Update():\n%s\nwant:\n%s", gj, wj)
			}
		})
	}
}
