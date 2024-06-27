package querybuilder

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type thing struct {
	ThingID     string    `bson:"thingID"`
	Active      bool      `bson:"active"`
	Ordinal     int       `bson:"ordinal"`
	Name        *string   `bson:"name"`
	TestNil     *int      `bson:"testNil"`
	NotInSchema string    `bson:"notInSchema"`
	Created     time.Time `bson:"created"`
	Attributes  []string  `bson:"attributes,omitempty"`
	SubThing    *subThing `bson:"sub,omitempty"`
}

type subThing struct {
	SubThingID string    `json:"subThingID"`
	Name       string    `json:"name"`
	Created    time.Time `json:"created"`
	Attributes *[]string `json:"attributes,omitempty"`
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
			"active": {
				"bsonType":    "bool",
				"description": "active status of the thing"
			},
			"ordinal": {
				"bsonType":    "number",
				"description": "ordinal value for the thing"
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

func Test_NewUpdateBuilder(t *testing.T) {
	type args struct {
		clctn  string
		schema string
	}
	tests := []struct {
		name string
		args args
		want *UpdateBuilder
	}{
		{
			"should create a new update builder",
			args{
				clctn:  "things",
				schema: thingsSchema,
			},
			&UpdateBuilder{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
				opts:  UpdateOptions(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUpdateBuilder(tt.args.clctn, tt.args.schema); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUpdateBuilder() = \n%v\n, want \n%v", got, tt.want)
			}
		})
	}
}

func TestUpdateBuilder_Update(t *testing.T) {
	var thng string = "thing"
	type fields struct {
		clctn string
		flds  map[string]string
		opts  *updateOptions
	}
	type args struct {
		doc  any
		opts []*updateOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.D
		wantErr bool
	}{
		{
			"should error if doc is not a struct",
			fields{},
			args{
				doc: "testing",
			},
			bson.D{},
			true,
		},
		{
			"should handle when doc is a pointer",
			fields{},
			args{
				doc: &thing{
					ThingID: thng,
				},
			},
			bson.D{bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "thing",
				}, bson.E{
					Key:   "active",
					Value: false,
				}}}},
			false,
		},
		{
			"should error when struct includes field that is not in the schema",
			fields{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
			},
			args{
				doc: thing{
					ThingID:     "123",
					Active:      true,
					Ordinal:     100,
					Name:        &thng,
					NotInSchema: "not in schema",
					Created:     time.Now(),
					Attributes:  []string{"tag1", "tag2"},
				},
				opts: []*updateOptions{
					UpdateOptions().SetStrictValidation(true),
				},
			},
			bson.D{},
			true,
		},
		{
			"should error when struct includes field that is not in the schema (option set on builder)",
			fields{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
				opts:  UpdateOptions().SetStrictValidation(true),
			},
			args{
				doc: thing{
					ThingID:     "123",
					Active:      true,
					Ordinal:     100,
					Name:        &thng,
					NotInSchema: "not in schema",
					Created:     time.Now(),
					Attributes:  []string{"tag1", "tag2"},
				},
			},
			bson.D{},
			true,
		},
		{
			"should create update document from struct",
			fields{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
			},
			args{
				doc: thing{
					ThingID:     "123",
					Active:      true,
					Ordinal:     100,
					Name:        &thng,
					NotInSchema: "not in schema",
					Created:     time.Now(),
					Attributes:  []string{"tag1", "tag2"},
					SubThing: &subThing{
						SubThingID: "456",
						Name:       "subthing",
						Created:    time.Now(),
						Attributes: &[]string{"tag3", "tag4"},
					},
				},
			},
			bson.D{bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "123",
				}, bson.E{
					Key:   "active",
					Value: true,
				}, bson.E{
					Key:   "ordinal",
					Value: 100,
				}, bson.E{
					Key:   "name",
					Value: "thing",
				}, bson.E{
					Key:   "notInSchema",
					Value: "not in schema",
				}, bson.E{
					Key:   "created",
					Value: time.Now(),
				}, bson.E{
					Key:   "attributes",
					Value: bson.A{"tag1", "tag2"},
				}, bson.E{
					Key:   "sub.subThingID",
					Value: "456",
				}, bson.E{
					Key:   "sub.name",
					Value: "subthing",
				}, bson.E{
					Key:   "sub.created",
					Value: time.Now(),
				}, bson.E{
					Key:   "sub.attributes",
					Value: bson.A{"tag3", "tag4"},
				}}}},
			false,
		},
		{
			"should create update document from struct with unset when empty",
			fields{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
			},
			args{
				doc: thing{
					ThingID: "123",
					Active:  true,
					Created: time.Now(),
				},
				opts: []*updateOptions{
					UpdateOptions().
						SetUnsetWhenEmpty("name", true).
						SetUnsetWhenEmpty("ordinal", true).
						SetUnsetWhenEmpty("attributes", true),
				},
			},
			bson.D{bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "123",
				}, bson.E{
					Key:   "active",
					Value: true,
				}, bson.E{
					Key:   "created",
					Value: time.Now(),
				}}}, bson.E{
				Key: "$unset",
				Value: bson.D{bson.E{
					Key:   "ordinal",
					Value: "",
				}, bson.E{
					Key:   "name",
					Value: "",
				}, bson.E{
					Key:   "attributes",
					Value: "",
				}}},
			},
			false,
		},
		{
			"should ignore specified fields in the update",
			fields{},
			args{
				doc: thing{
					ThingID:     "123",
					Active:      true,
					Name:        &thng,
					NotInSchema: "not in schema",
					Created:     time.Now(),
				},
				opts: []*updateOptions{
					UpdateOptions().SetIgnoreFields("name", "notInSchema"),
				},
			},
			bson.D{bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "123",
				}, bson.E{
					Key:   "active",
					Value: true,
				}, bson.E{
					Key:   "created",
					Value: time.Now(),
				}}},
			},
			false,
		},
		{
			"should properly addToSet fields when specified",
			fields{},
			args{
				doc: thing{
					ThingID:    "123",
					Active:     true,
					Attributes: []string{"tag1", "tag2"},
				},
				opts: []*updateOptions{
					UpdateOptions().SetAddToSet("attributes", true),
				},
			},
			bson.D{bson.E{
				Key: "$addToSet",
				Value: bson.D{bson.E{
					Key: "attributes",
					Value: bson.D{bson.E{
						Key:   "$each",
						Value: bson.A{"tag1", "tag2"},
					}},
				}},
			}, bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "123",
				}, bson.E{
					Key:   "active",
					Value: true,
				}},
			}},
			false,
		},
		{
			"options provided to Update should override any default options set on the builder",
			fields{
				clctn: "things",
				flds:  parseSchema(thingsSchema),
				opts:  UpdateOptions().SetStrictValidation(true),
			},
			args{
				doc: thing{
					ThingID:     "123",
					Active:      true,
					Ordinal:     100,
					Name:        &thng,
					NotInSchema: "not in schema",
					Created:     time.Now(),
					Attributes:  []string{"tag1", "tag2"},
				},
				opts: []*updateOptions{
					UpdateOptions().SetStrictValidation(false),
				},
			},
			bson.D{bson.E{
				Key: "$set",
				Value: bson.D{bson.E{
					Key:   "thingID",
					Value: "123",
				}, bson.E{
					Key:   "active",
					Value: true,
				}, bson.E{
					Key:   "ordinal",
					Value: 100,
				}, bson.E{
					Key:   "name",
					Value: "thing",
				}, bson.E{
					Key:   "notInSchema",
					Value: "not in schema",
				}, bson.E{
					Key:   "created",
					Value: time.Now(),
				}, bson.E{
					Key:   "attributes",
					Value: bson.A{"tag1", "tag2"},
				}}}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ub := &UpdateBuilder{
				clctn: tt.fields.clctn,
				flds:  tt.fields.flds,
				opts:  tt.fields.opts,
			}

			got, err := ub.Update(tt.args.doc, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateBuilder.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gj, _ := bson.MarshalExtJSONIndent(got, false, false, "", "  ")
			wj, _ := bson.MarshalExtJSONIndent(tt.want, false, false, "", "  ")

			if !reflect.DeepEqual(gj, wj) {
				t.Errorf("UpdateBuilder.Update():\n%s\nwant:\n%s", gj, wj)
			}
		})
	}
}
