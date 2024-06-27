package querybuilder

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type UpdateBuilder struct {
	collection       string
	fieldTypes       map[string]string
	strictValidation bool
}

func NewUpdateBuilder(collection string, schema any, strictValidation ...bool) *UpdateBuilder {
	ub := UpdateBuilder{
		collection:       collection,
		fieldTypes:       parseSchema(schema),
		strictValidation: false,
	}

	// override strict validation if provided
	if len(strictValidation) > 0 {
		ub.strictValidation = strictValidation[0]
	}

	return &ub
}

func (ub *UpdateBuilder) Update(doc any, preserveSet ...bool) (bson.D, error) {
	// create the update document
	set := bson.D{}
	update := bson.D{}

	// check for preserve set
	ats := bson.D{}
	ps := false
	if len(preserveSet) > 0 && preserveSet[0] {
		ps = true
	}

	// parse each field in the doc and validate against the schema
	v := reflect.ValueOf(doc)

	// if the doc is a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// ensure the doc is a struct (nothing to build an update for otherwise)
	if v.Kind() != reflect.Struct {
		return update, fmt.Errorf("doc must be a struct")
	}

	var proc func(reflect.Value, string) error
	proc = func(v reflect.Value, prefix string) error {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Struct {
			tp := v.Type()
			vf := reflect.VisibleFields(tp)
			for _, f := range vf {
				fv := v.FieldByIndex(f.Index)
				ft := f.Type

				// determine to the field name
				fn := ub.getMongoFieldName(f)
				if prefix != "" {
					fn = strings.Join([]string{prefix, fn}, ".")
				}

				// skip empty field names (no way to build a path to the field)
				if fn == "" {
					continue
				}

				// if the field is a pointer, dereference it
				if ft.Kind() == reflect.Ptr {
					ft = ft.Elem()
				}

				// detect if the field is a type of struct
				// we need to recurse into in order to build the update
				skip := false
				for _, nm := range []string{
					"Time",
				} {
					if nm == ft.Name() {
						skip = true
					}
				}

				// check to see if the field is a struct
				if !skip && ft.Kind() == reflect.Struct {
					proc(fv, fn)
					continue
				}

				// check to see if the field is in the schema
				if _, ok := ub.fieldTypes[fn]; !ok {
					if ub.strictValidation {
						return fmt.Errorf("field %s does not exist in collection %s", fn, ub.collection)
					}

					continue
				}

				// add the field name and value to the set document
				set = append(set, bson.E{
					Key:   fn,
					Value: ub.getValue(fv),
				})
			}
		}

		return nil
	}

	// iterate each field on the type
	if err := proc(v, ""); err != nil {
		return update, err
	}

	// add the set document to the update document
	update = append(update, bson.E{
		Key:   "$set",
		Value: set,
	})

	// add the preserve set document to the update document
	// if applicable
	if ps {
		update = append(update, bson.E{
			Key:   "$addToSet",
			Value: ats,
		})
	}

	return update, nil
}

func (ub *UpdateBuilder) getMongoFieldName(field reflect.StructField) string {
	// pull the bson tag from the field
	tag := field.Tag.Get("bson")

	// fallback to json tag if bson tag is empty
	if tag == "" {
		tag = field.Tag.Get("json")
	}

	// strip out any omitempty or other flags
	tag = strings.Split(tag, ",")[0]

	return tag
}

func (ub *UpdateBuilder) getValue(value reflect.Value) any {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	// if the field is a Time, convert to a string
	if value.Type().Name() == "Time" {
		return value.Interface().(time.Time).Format(time.RFC3339)
	}

	return value.Interface()
}
