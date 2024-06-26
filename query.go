package querybuilder

import (
	"fmt"
	"strconv"

	queryoptions "go.jtlabs.io/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QueryBuilder is a type that makes working with Mongo driver Find methods easier
// when used in combination with a QueryOptions struct that specifies filters,
// pagination details, sorting instructions and field projection details.
type QueryBuilder struct {
	collection       string
	fieldTypes       map[string]string
	strictValidation bool
}

// NewQueryBuilder returns a new instance of a QueryBuilder object for constructing
// filters and options suitable for use with Mongo driver Find methods
func NewQueryBuilder(collection string, schema any, strictValidation ...bool) *QueryBuilder {
	qb := QueryBuilder{
		collection:       collection,
		fieldTypes:       map[string]string{},
		strictValidation: false,
	}

	// parse the schema
	if schema != nil {
		// look for a map[string]any as the schema
		if s, ok := schema.(map[string]any); ok {
			qb.fieldTypes = parseMapSchema(s)
		}

		// look for a bson.M as the schema
		if s, ok := schema.(bson.M); ok {
			qb.fieldTypes = parseBSONSchema(s)
		}

		// look for a []bit (marshalled JSON) as the schema
		if s, ok := schema.([]byte); ok {
			qb.fieldTypes = parseJSONSchema(s)
		}

		// look for a string (serialized JSON) as the schema
		if s, ok := schema.(string); ok {
			qb.fieldTypes = parseStringSchema(s)
		}
	}

	// override strict validation if provided
	if len(strictValidation) > 0 {
		qb.strictValidation = strictValidation[0]
	}

	return &qb
}

// Filter builds a suitable bson document to send to any of the find methods
// exposed by the Mongo driver. This method can validate the provided query
// options against the schema that was used to build the QueryBuilder instance
// when the QueryBuilder has strict validation enabled.
//
// The supported bson types for filter/search are:
// * array (strings only and not with $in operator unless sub items are strings)
// * bool
// * date
// * decimal
// * double
// * int
// * long
// * object (field detection)
// * string
// * timestamp
//
// The non-supported bson types for filter/search at this time
// * object (actual object comparison... only fields within the object are supported)
// * array (non string data)
// * binData
// * objectId
// * null
// * regex
// * dbPointer
// * javascript
// * symbol
// * javascriptWithScope
// * minKey
// * maxKey
func (qb QueryBuilder) Filter(qo queryoptions.Options, o ...LogicalOperator) (bson.M, error) {
	filter := bson.M{}
	oper := And

	if len(o) > 0 {
		oper = o[0]
	}

	if len(qo.Filter) > 0 {
		for field, values := range qo.Filter {
			var bsonType string

			// lookup the field
			if bt, ok := qb.fieldTypes[field]; ok {
				bsonType = bt
			}

			// check for strict field validation
			if bsonType == "" && qb.strictValidation {
				return nil, fmt.Errorf("field %s does not exist in collection %s", field, qb.collection)
			}

			switch bsonType {
			case "array", "object", "string":
				f := detectStringComparisonOperator(field, values, bsonType)
				filter = combine(filter, f)
			case "bool":
				for _, value := range values {
					bv, _ := strconv.ParseBool(value)
					f := primitive.M{field: bv}
					filter = combine(filter, f)
				}
			case "date", "timestamp":
				f := detectDateComparisonOperator(field, values, oper)
				filter = combine(filter, f)
			case "decimal", "double", "int", "long":
				f := detectNumericComparisonOperator(field, values, bsonType, oper)
				filter = combine(filter, f)
			}
		}
	}

	return filter, nil
}

// FindOptions creates a mongo.FindOptions struct with pagination details, sorting,
// and field projection instructions set as specified in the query options input
func (qb QueryBuilder) FindOptions(qo queryoptions.Options) (*options.FindOptions, error) {
	opts := options.Find()

	// determine pagination for the options
	qb.setPaginationOptions(qo.Page, opts)

	// determine projection for the options
	if err := qb.setProjectionOptions(qo.Fields, opts); err != nil {
		return nil, err
	}

	// determine sorting for the options
	if err := qb.setSortOptions(qo.Sort, opts); err != nil {
		return nil, err
	}

	return opts, nil
}

func (qb QueryBuilder) setPaginationOptions(pagination map[string]int, opts *options.FindOptions) {
	// check for limit
	if limit, ok := pagination["limit"]; ok {
		opts.SetLimit(int64(limit))

		// check for offset (once limit is set)
		if offset, ok := pagination["offset"]; ok {
			opts.SetSkip(int64(offset))
		}

		// check for skip (once limit is set)
		if skip, ok := pagination["skip"]; ok {
			opts.SetSkip(int64(skip))
		}
	}

	// check for page and size
	if size, ok := pagination["size"]; ok {
		opts.SetLimit(int64(size))

		// set skip (requires understanding of size)
		if page, ok := pagination["page"]; ok {
			opts.SetSkip(int64(page * size))
		}
	}
}

func (qb QueryBuilder) setProjectionOptions(fields []string, opts *options.FindOptions) error {
	// set field projections option
	if len(fields) > 0 {
		prj := map[string]int{}
		for _, field := range fields {
			val := 1

			// handle when the first char is a - (don't display field in result)
			if field[0:1] == "-" {
				field = field[1:]
				val = 0
			}

			// handle scenarios where the first char is a + (redundant)
			if len(field) > 0 && field[0:1] == "+" {
				field = field[1:]
			}

			// lookup field in the fieldTypes dictionary if strictValidation is true
			if qb.strictValidation {
				if _, ok := qb.fieldTypes[field]; !ok {
					// we have a problem
					return fmt.Errorf("field %s does not exist in collection %s", field, qb.collection)
				}
			}

			// add the field to the project dictionary
			prj[field] = val
		}

		// add the projection to the FindOptions
		if len(prj) > 0 {
			opts.SetProjection(prj)
		}
	}

	return nil
}

func (qb QueryBuilder) setSortOptions(fields []string, opts *options.FindOptions) error {
	if len(fields) > 0 {
		sort := map[string]int{}
		for _, field := range fields {
			val := 1

			if field[0:1] == "-" {
				field = field[1:]
				val = -1
			}

			if field[0:1] == "+" {
				field = field[1:]
			}

			// lookup field in the fieldTypes dictionary if strictValidation is true
			if qb.strictValidation {
				if _, ok := qb.fieldTypes[field]; !ok {
					// we have a problem
					return fmt.Errorf("field %s does not exist in collection %s", field, qb.collection)
				}
			}

			sort[field] = val
		}

		opts.SetSort(sort)
	}

	return nil
}
