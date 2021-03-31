package mongo

import (
	"reflect"
	"testing"
	"time"

	"github.com/brozeph/queryoptions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func TestQueryBuilder_Filter(t *testing.T) {
	type fields struct {
		collection       string
		fieldTypes       map[string]string
		strictValidation bool
	}
	type args struct {
		qs string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bson.D
		wantErr bool
	}{
		{
			name: "test with no query args",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qs: "",
			},
			want:    bson.D{},
			wantErr: false,
		},
		{
			name: "should error with strict validation and mismatched field",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: true,
			},
			args: args{
				qs: "filter[nofield]=error",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should not error without strict validation and mismatched field",
			fields: fields{
				collection:       "test",
				fieldTypes:       map[string]string{},
				strictValidation: false,
			},
			args: args{
				qs: "filter[nofield]=error",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "should properly detect and type numeric values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"deVal": "decimal",
					"doVal": "double",
					"iVal":  "int",
					"lVal":  "long",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[doVal]=0.000000000000000000000000000000009&filter[deVal]=10.01&filter[iVal]=2147483647&filter[lVal]=9223372036854775807",
			},
			want: bson.D{
				primitive.E{
					Key:   "deVal",
					Value: float32(10.01),
				},
				primitive.E{
					Key:   "doVal",
					Value: float64(0.000000000000000000000000000000009),
				},
				primitive.E{
					Key:   "iVal",
					Value: int32(2147483647),
				},
				primitive.E{
					Key:   "lVal",
					Value: int64(9223372036854775807),
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric values with $in operator",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
					"iVal2": "decimal",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=1,2,3,4,5&filter[iVal2]=1.1,2.2,3.3",
			},
			want: bson.D{
				primitive.E{
					Key: "iVal1",
					Value: primitive.E{
						Key:   "$in",
						Value: primitive.A{int32(1), int32(2), int32(3), int32(4), int32(5)},
					},
				},
				primitive.E{
					Key: "iVal2",
					Value: primitive.E{
						Key:   "$in",
						Value: primitive.A{float32(1.1), float32(2.2), float32(3.3)},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle numeric operators (lt, lte, gt, gte, ne)",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"iVal1": "int",
					"iVal2": "int",
					"iVal3": "int",
					"iVal4": "int",
					"iVal5": "int",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[iVal1]=%3C4&filter[iVal2]=%3C%3D3&filter[iVal3]=%3E1&filter[iVal4]=%3E%3D2&filter[iVal5]=%21%3D5",
			},
			want: bson.D{
				primitive.E{
					Key: "iVal1",
					Value: bson.D{primitive.E{
						Key:   "$lt",
						Value: int32(4),
					}},
				},
				primitive.E{
					Key: "iVal2",
					Value: bson.D{primitive.E{
						Key:   "$lte",
						Value: int32(3),
					}},
				},
				primitive.E{
					Key: "iVal3",
					Value: bson.D{primitive.E{
						Key:   "$gt",
						Value: int32(1),
					}},
				},
				primitive.E{
					Key: "iVal4",
					Value: bson.D{primitive.E{
						Key:   "$gte",
						Value: int32(2),
					}},
				},
				primitive.E{
					Key: "iVal5",
					Value: bson.D{primitive.E{
						Key:   "$ne",
						Value: int32(5),
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle bool types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"bVal1": "bool",
					"bVal2": "bool",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[bVal1]=true&filter[bVal2]=false",
			},
			want: bson.D{
				primitive.E{
					Key:   "bVal1",
					Value: true,
				},
				primitive.E{
					Key:   "bVal2",
					Value: false,
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle date types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "date",
					"dVal2": "date",
					"dVal3": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=2020-01-01T12:00:00.000Z&filter[dVal2]=2021-02-16T02:04:05.000Z&filter[dVal3]=2021-02-16T02:04:05.000Z,2020-01-01T12:00:00.000Z",
			},
			want: bson.D{
				primitive.E{
					Key:   "dVal1",
					Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
				},
				primitive.E{
					Key:   "dVal2",
					Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
				},
				primitive.E{
					Key: "dVal3",
					Value: primitive.E{
						Key:   "$in",
						Value: primitive.A{time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC), time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC)},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle operators on date types",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"dVal1": "date",
					"dVal2": "date",
					"dVal3": "date",
					"dVal4": "date",
					"dVal5": "date",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[dVal1]=<2020-01-01T12:00:00.000Z&filter[dVal2]=<=2021-02-16T02:04:05.000Z&filter[dVal3]=>2021-02-16T02:04:05.000Z&filter[dVal4]=>=2021-02-16T02:04:05.000Z&filter[dVal5]=!=2020-01-01T12:00:00.000Z",
			},
			want: bson.D{
				primitive.E{
					Key: "dVal1",
					Value: bson.D{primitive.E{
						Key:   "$lt",
						Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
					}},
				},
				primitive.E{
					Key: "dVal2",
					Value: bson.D{primitive.E{
						Key:   "$lte",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					}},
				},
				primitive.E{
					Key: "dVal3",
					Value: bson.D{primitive.E{
						Key:   "$gt",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					}},
				},
				primitive.E{
					Key: "dVal4",
					Value: bson.D{primitive.E{
						Key:   "$gte",
						Value: time.Date(2021, time.February, 16, 2, 4, 5, 0, time.UTC),
					}},
				},
				primitive.E{
					Key: "dVal5",
					Value: bson.D{primitive.E{
						Key:   "$ne",
						Value: time.Date(2020, time.January, 1, 12, 0, 0, 0, time.UTC),
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string type using $exists operator with object fields",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"oVal":       "object",
					"oVal.sVal1": "string",
					"oVal.sVal2": "string",
					"oVal.sVal3": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[oVal]=sVal1,!=sVal2,-sVal3",
			},
			want: bson.D{
				primitive.E{
					Key: "oVal.sVal1",
					Value: primitive.E{
						Key:   "$exists",
						Value: true,
					},
				},
				primitive.E{
					Key: "oVal.sVal2",
					Value: primitive.E{
						Key:   "$exists",
						Value: false,
					},
				},
				primitive.E{
					Key: "oVal.sVal3",
					Value: primitive.E{
						Key:   "$exists",
						Value: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string type using $in operator with array of values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=value1,value2,value3",
			},
			want: bson.D{
				primitive.E{
					Key: "sVal1",
					Value: primitive.E{
						Key:   "$in",
						Value: primitive.A{"value1", "value2", "value3"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle array type and not use $in operator with array of values",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"aVal1": "array",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[aVal1]=value1,value2,value3",
			},
			want: bson.D{
				primitive.E{
					Key:   "aVal1",
					Value: primitive.A{"value1", "value2", "value3"},
				},
			},
			wantErr: false,
		},
		{
			name: "should properly handle string wildcards with regexes",
			fields: fields{
				collection: "test",
				fieldTypes: map[string]string{
					"sVal1": "string",
					"sVal2": "string",
					"sVal3": "string",
					"sVal4": "string",
					"sVal5": "string",
				},
				strictValidation: false,
			},
			args: args{
				qs: "filter[sVal1]=*value&filter[sVal2]=value*&filter[sVal3]=*value*&filter[sVal4]=value&filter[sVal5]=!=value",
			},
			want: bson.D{
				primitive.E{
					Key: "sVal1",
					Value: primitive.Regex{
						Pattern: "value$",
						Options: "i",
					},
				},
				primitive.E{
					Key: "sVal2",
					Value: primitive.Regex{
						Pattern: "^value",
						Options: "i",
					},
				},
				primitive.E{
					Key: "sVal3",
					Value: primitive.Regex{
						Pattern: "value",
						Options: "i",
					},
				},
				primitive.E{
					Key:   "sVal4",
					Value: "value",
				},
				primitive.E{
					Key: "sVal5",
					Value: primitive.E{
						Key:   "$ne",
						Value: "value",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := QueryBuilder{
				collection:       tt.fields.collection,
				fieldTypes:       tt.fields.fieldTypes,
				strictValidation: tt.fields.strictValidation,
			}

			qo, err := queryoptions.FromQuerystring(tt.args.qs)
			if err != nil {
				t.Errorf("QueryOptions.FromQuerystring() error = %v", err)
				return
			}

			got, err := qb.Filter(qo)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryBuilder.Filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// check the length of what was returned and what is expected...
			if len(got) != len(tt.want) {
				t.Errorf("QueryBuilder.Filter() = %v, want %v", got, tt.want)
			}

			// convert what was returned into a Map for lookup
			gotMap := map[string]primitive.E{}
			for _, e := range got {
				gotMap[e.Key] = e
			}

			// iterate through the keys of what is wanted to ensure each key value matches
			for _, e := range tt.want {
				if val, ok := gotMap[e.Key]; ok {
					if val.Key != e.Key || !reflect.DeepEqual(val.Value, e.Value) {
						// values do not match
						t.Errorf("QueryBuilder.Filter() = %v, want %v", val, e)
					}

					continue
				}

				// map was missing the key...
				t.Errorf("QueryBuilder.Filter() = missing field %v in response (%v)", e.Key, got)
			}
		})
	}
}
