package querybuilder

import (
	"fmt"
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
)

type UpdateBuilder struct {
	clctn string
	flds  map[string]string
	opts  *updateOptions
}

// NewUpdateBuilder creates a new instance of an UpdateBuilder object for constructing
// update documents suitable for use with the Mongo driver Update methods.
func NewUpdateBuilder(collection string, schema any, opts ...*updateOptions) *UpdateBuilder {
	ub := UpdateBuilder{
		clctn: collection,
		flds:  parseSchema(schema),
		opts:  mergeUpdateOptions(opts...),
	}

	return &ub
}

// Update creates a suitable bson document to send to any of the update methods
// exposed by the Mongo driver. This method supports optional additional options
// that can be used to control the behavior of the update document. Any options
// provided will override the default options set on the UpdateBuilder instance.
func (ub *UpdateBuilder) Update(doc any, opts ...*updateOptions) (bson.D, error) {
	// create the update document and it's components
	ats := bson.D{}
	set := bson.D{}
	us := bson.D{}
	upd := bson.D{}
	uo := mergeUpdateOptions(ub.opts, mergeUpdateOptions(opts...))

	// parse each field in the doc and validate against the schema
	v := reflect.ValueOf(doc)

	// if the doc is a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// ensure the doc is a struct (nothing to build an update for otherwise)
	if v.Kind() != reflect.Struct {
		return upd, fmt.Errorf("doc must be a struct")
	}

	// parse each field in the doc...
	if err := forEachField(v, "", func(pth string, val any) error {
		// when strict validation is requested, check for fields present on the doc
		// but not in the schema
		if uo.strictValidation != nil && *uo.strictValidation {
			if _, ok := ub.flds[pth]; !ok {
				return fmt.Errorf("field %s does not exist in collection %s", pth, ub.clctn)
			}
		}

		// check for unset fields
		if isValueEmpty(val) {
			if b, ok := uo.unsetWhenEmpty[pth]; ok && b {
				us = append(us, bson.E{
					Key:   pth,
					Value: "",
				})
			}

			return nil
		}

		// check for ignored fields
		if uo.fieldIgnored(pth) {
			return nil
		}

		// check for addToSet fields
		if b, ok := uo.addToSet[pth]; ok && b {
			ats = append(ats, bson.E{
				Key: pth,
				Value: bson.D{bson.E{
					Key:   "$each",
					Value: val,
				}},
			})

			return nil
		}

		// add the field name and value to the set document
		set = append(set, bson.E{
			Key:   pth,
			Value: val,
		})

		return nil
	}); err != nil {
		return upd, err
	}

	// add the addToSet document to the update document
	if len(ats) > 0 {
		upd = append(upd, bson.E{
			Key:   "$addToSet",
			Value: ats,
		})
	}

	// add the set document to the update document
	if len(set) > 0 {
		upd = append(upd, bson.E{
			Key:   "$set",
			Value: set,
		})
	}

	// add the unset document to the update document
	if len(us) > 0 {
		upd = append(upd, bson.E{
			Key:   "$unset",
			Value: us,
		})
	}

	return upd, nil
}
