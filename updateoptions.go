package querybuilder

type updateOptions struct {
	addToSet         map[string]bool
	ignoreFields     []string
	strictValidation *bool
	unsetWhenEmpty   map[string]bool
}

// UpdateOptions provides a set of options for the UpdateBuilder.
func UpdateOptions() *updateOptions {
	return &updateOptions{}
}

// SetAddToSet instructs the updater to use $addToSet instead of $set
// when the field type is a slice, and the field name matches the provided field.
//
//	func example() {
//		schema := `{
//			"$jsonSchema": {
//				"bsonType": "object",
//					"properties": {
//						"tagList": {
//							"bsonType": "array",
//							"description": "list of tags",
//							"items": {
//								"bsonType": "string"
//							}
//						},
//						"authorList": {
//							"bsonType": "array",
//							"description": "list of authors",
//							"items": {
//								"bsonType": "object",
//								"properties": {
//									"name": {
//										"bsonType": "string",
//										"description": "name of the author"
//									},
//									"email": {
//										"bsonType": "string",
//										"description": "email of the author"
//									}
//								}
//							}
//						}
//					}
//				}
//			}`
//
//		// create an update builder
//		ub := NewUpdateBuilder("collection", schema)
//
//		// create an update document that uses $addToSet for tagList (but use $set for authorList)
//		opts := UpdateOptions().SetAddToSet("tagList", true)
//
//		// retrieve the update document
//		doc := article{
//			TagList: []string{"new", "tag"},
//			AuthorList: []author{
//				{
//					Name:  "John Doe",
//					Email: "joh@n.do.e",
//				},
//			},
//		}
//
//		// do something with the update document...
//		// which looks something like this:
//		// bson.D{
//		//   {"$addToSet", bson.D{
//		//     {"tagList", bson.D{
//		//       {"$each", []string{"new", "tag"}},
//		//     }},
//		//   }},
//		//   {"$set", bson.D{
//		//     {"authorList", bson.A[]{
//		//       bson.D{
//		//         {"name", "John Doe"},
//		//         {"email", "joh@n.do.e",
//		//       },
//		//     }},
//		//   }},
//		// }
//		update, err := ub.Update(doc, opts)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//	}
func (uo *updateOptions) SetAddToSet(fld string, b bool) *updateOptions {
	if uo.addToSet == nil {
		uo.addToSet = map[string]bool{}
	}
	uo.addToSet[fld] = b
	return uo
}

// SetIgnoreFields instructs the updater to ignore the provided fields when building the update document.
func (uo *updateOptions) SetIgnoreFields(flds ...string) *updateOptions {
	uo.ignoreFields = append(uo.ignoreFields, flds...)
	return uo
}

// SetStrictValidation instructs the updater to validate the provided document against the schema.
// If the document provided does not match the schema, the updater will return an error.
func (uo *updateOptions) SetStrictValidation(b bool) *updateOptions {
	uo.strictValidation = &b
	return uo
}

// SetUnsetWhenNil instructs the updater to use $unset when a field value is considered
// empty. This is helpful when a struct that has a field that is set to `nil` (or if it is
// another type with a zero value, like time.Time, a number, or a string, etc.). By default
// if the value is empty, the updater will not include the field in the update document.
//
//	func example() {
//		schema := `{
//			"$jsonSchema": {
//				"bsonType": "object",
//				"properties": {
//					"thingID": {
//						"bsonType":    "number",
//						"description": "number of the thing"
//					},
//					"created": {
//						"bsonType":    "date",
//						"description": "time at which the thing was created"
//					},
//					"name": {
//						"bsonType":    "string",
//						"description": "name of the thing"
//					}
//				}
//			}
//		}`
//
//		// create an update builder
//		ub := NewUpdateBuilder("collection", schema)
//
//		// create an update document that uses $unset for created when the value is empty
//		opts := UpdateOptions().SetUnsetWhenEmpty("created", true)
//
//		// retrieve the update document
//		doc := thing{
//			ThingID: 123,
//			Name:    "",
//		}
//
//		// do something with the update document...
//		// which looks something like this:
//		// bson.D{
//		//   {"$set", bson.D{
//		//     {"thingID", 123},
//		//   }},
//		//   {"$unset", bson.D{
//		//     {"created", ""},
//		//   }},
//		// }
//		// notice that "name" is simply not included in the update document
//		// because it is empty and not explicitly set as an unset when empty
//		// field
//		update, err := ub.Update(doc, opts)
//		if err != nil {
//			fmt.Println(err)
//			return
//		}
//	}
func (uo *updateOptions) SetUnsetWhenEmpty(fld string, b bool) *updateOptions {
	if uo.unsetWhenEmpty == nil {
		uo.unsetWhenEmpty = map[string]bool{}
	}
	uo.unsetWhenEmpty[fld] = b
	return uo
}

func (uo *updateOptions) fieldIgnored(fld string) bool {
	for _, f := range uo.ignoreFields {
		if f == fld {
			return true
		}
	}

	return false
}

func mergeUpdateOptions(opts ...*updateOptions) *updateOptions {
	uo := UpdateOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		for fld, b := range opt.addToSet {
			uo.SetAddToSet(fld, b)
		}

		uo.SetIgnoreFields(opt.ignoreFields...)

		if opt.strictValidation != nil {
			uo.SetStrictValidation(*opt.strictValidation)
		}

		for fld, b := range opt.unsetWhenEmpty {
			uo.SetUnsetWhenEmpty(fld, b)
		}
	}

	return uo
}
