# MongoDB QueryBuilder

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/jtlabsio/mongo) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/brozeph/mongoquerybuilder/main/LICENSE) [![Coverage](http://gocover.io/_badge/github.com/jtlabsio/mongo)](http://gocover.io/github.com/jtlabsio/mongo)


This library exists to ease the creation of MongoDB filter and FindOptions structs when using the MongoDB driver in combination with a [JSONAPI query parser](https://github.com/jtlabsio/query).

## Installation

```bash
go get -u go.jtlabs.io/mongo
```

## Usage

Example code below translated from [examples/examples.go](examples/examples.go) - for more info, see the example file run running instructions.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	mongobuilder "go.jtlabs.io/mongo"
	queryoptions "go.jtlabs.io/query"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// schema for things collection (used by mongo query builder)
var thingsSchema = bson.M{
	"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": []string{"thingID"},
		"properties": bson.M{
			"thingID": bson.M{
				"bsonType":    "string",
				"description": "primary identifier for the thing",
			},
			"created": bson.M{
				"bsonType":    "date",
				"description": "time at which the thing was created",
			},
			"name": bson.M{
				"bsonType":    "string",
				"description": "name of the thing",
			},
			"types": bson.M{
				"bsonType":    "array",
				"description": "type tags for the thing",
			},
		},
	},
}

// create a new MongoDB QueryBuilder (with strict validation set to true)
var builder = querybuilder.NewQueryBuilder("things", thingsSchema, true)

//pointer for the mongo collection to query from
var collection *mongo.Collection

func getThings(w http.ResponseWriter, r *http.Request) {
	opt, err := queryoptions.FromQuerystring(r.URL.RawQuery)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// build a bson.M filter for the Find based on queryoptions filters
	filter, err := builder.Filter(opt)
	if err != nil {
		// NOTE: will only error when strictValidation is true
		fmt.Fprint(w, err)
		return
	}

	// build options (pagination, sorting, field projection) based on queryoptions
	fo, err := builder.FindOptions(opt)
	if err != nil {
		// NOTE: will only error when strictValidation is true
		fmt.Fprint(w, err)
		return
	}

	// now use the filter and options in a Find call to the Mongo collection
	cur, err := collection.Find(context.TODO(), filter, fo)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	defer cur.Close(context.TODO())

	data := []struct {
		ThingID string    `bson:"thingID"`
		Name    string    `bson:"name"`
		Created time.Time `bson:"created"`
		Types   []string  `bson:"types"`
	}{}
	if err = cur.All(context.TODO(), &data); err != nil {
		fmt.Fprint(w, err)
		return
	}

	re, _ := json.Marshal(data)
	fmt.Fprint(w, string(re))
}

func main() {
	// create a MongoDB client
	mc, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// connect to MongoDB
	if err := mc.Connect(context.TODO()); err != nil {
		log.Fatal(err)
	}

	// create a collection with the schema
	colOpts := options.CreateCollection().SetValidator(thingsSchema)
	mc.Database("things-db").CreateCollection(context.TODO(), "things", colOpts)

	// set the collection pointer
	collection = mc.Database("things-db").Collection("things")

	http.HandleFunc("/v1/things", getThings)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### QueryBuilder

A `QueryBuilder` struct can be created per MongoDB collection and will look to a [JSONSchema](https://docs.mongodb.com/manual/reference/operator/query/jsonSchema/) defined (and often used as a validator) for the MongoDB collection to build queries and propery coerce types for parameters that are provided in a [JSON API Query Options](https://github.com/jtlabsio/query) object as filters.

#### NewQueryBuilder

New `QueryBuilder` instances can be created using the `NewQueryBuilder` function:

```go
jsonSchema := bson.D{ /* a JSON schema here... */ }
qb := querybuilder.NewQueryBuilder("collectionName", jsonSchema)
```

By default, the `QueryBuilder` does not perform strict schema validation when constructing filter instances and options for Find queries. Strict schema validation can be enabled which will result in an `error` when trying to build a filter referencing any fields that do not exist within the provided schema or when trying to sort or project based on fields that do not exist in the schema.

```go
// With strict validation enabled
jsonSchema := bson.D{ /* a JSON schema here... */ }
qb := querybuilder.NewQueryBuilder("collectionName", jsonSchema, true)
```

#### Filter

The filter method returns a `bson.M{}` that can be used for excuting Find operations in Mongo.

```go
func getAllThings(w http.ResponseWriter, r *http.Request) {
  // hydrate a QueryOptions index from the request
  opt, err := queryoptions.FromQuerystring(r.URL.RawQuery)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

  // a query filter in a bson.M based on QueryOptions filters
	f, _ := builder.Filter(opt)

	// options (pagination, sorting, field projection) based on QueryOptions
	fo, _ := builder.FindOptions(opt)

  // now use the filter and options in a Find call to the Mongo collection
	cur, err := collection.Find(context.TODO(), f, fo)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

  /* do cool stuff with the cursor... */
}
```

##### Filter

The `QueryOptions` (<https://github.com/jtlabsio/query>) package is great for parsing JSONAPI compliant `filter`, `fields`, `sort` and `page` details that are provided in the querystring of an API request. There are some nuances in the way in which filters are constructed based on the parameters.

```go
// a query filter in a bson.M based on QueryOptions Filter values
f, err := builder.Filter(opt)
if err != nil {
  // this only occurs when strict schema validation is true
  // and a field is named in the querystring that doesn't actually
  // exist as defined in the schema... this is NOT the default
  // behavior
}
```

###### Query Operators

*__note:__ to illustrate the concept for example purposes, the querystring samples shown below are not URL encoded, but should be under normal circumstances...*

*string bsonType*

For `string` bsonType fields in the schema, the following operators can be leveraged with specific querystring hints:

* `begins with` (i.e. `{ "name": { "regex": /^term/, "options": "i" } }`): `?filter[name]=term*`
* `ends with` (i.e. `{ "name": { "regex": /term$/, "options": "i" } }`): `?ilter[name]=*term`
* `exact match` (i.e. `{ "name": { "regex": /^term$/ } }`): `?filter[name]="term"`
* `not equal` (i.e. `{ "name": { "$ne": "term" } }`): `?filter[name]=!=term`
* `in` (i.e. `{ "name": { "$in": [ ... ] } }`): `?filter[name]=term1,term2,term3,term4`
* standard comparison (i.e. `{ "name": "term" }`): `?filter[name]=term`

*numeric bsonType*

For `numeric` bsonType fields in the schema (`int`, `long`, `decimal`, and `double`), any values provided in the querystring that are parsed by `QueryOptions` are coerced to the appropriate type when constructing the filter. Additionally, the following operators can be used in combination with querystring hints:

* `less than` (i.e. `{ "age": { "$lt": 5 } }`): `?filter[age]=<5`
* `less than equal` (i.e. `{ "age": { "$lte": 5 } }`): `?filter[age]=<=5`
* `greater than` (i.e. `{ "age": { "$gt": 5 } }`): `?filter[age]=>5`
* `greater than equal` (i.e. `{ "age": { "$gte": 5 } }`): `?filter[age]=>=5`
* `not equals` (i.e. `{ "age": { "$ne": 5 } }`): `?filter[age]=!=5`
* `in` (i.e. `{ "age": { "$in": [1,2,3,4,5] } }`): `?filter[age]=1,2,3,4,5`
* standard comparison (i.e. `{ "age": 5 }`): `?filter[age]=5`

*date bsonType*

For `date` bsonType fields in the schema (`date` and `timestamp`), any values in the querystring are converted according to `RFC3339` and used in the filter. The following operators can be used in combination with querystring hints:

* `less than` (i.e. `{ "someDate": { "$lt": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=<2021-02-16T02:04:05.000Z`
* `less than equal` (i.e. `{ "someDate": { "$lte": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=<=2021-02-16T02:04:05.000Z`
* `greater than` (i.e. `{ "someDate": { "$gt": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=>2021-02-16T02:04:05.000Z`
* `greater than equal` (i.e. `{ "someDate": { "$gte": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=>=2021-02-16T02:04:05.000Z`
* `not equals` (i.e. `{ "someDate": { "$ne": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=!=2021-02-16T02:04:05.000Z`
* `in` (i.e. `{ "someDate": { "$in": [ ... ] } }`): `?filter[someDate]=2021-02-16T00:00:00.000Z,2021-02-15T00:00:00.000Z`
* standard comparison (i.e. `{ "someDate": new Date("2021-02-16T02:04:05.000Z") }`): `?filter[someDate]=2021-02-16T02:04:05.000Z`

#### FindOptions

Pagination, sorting and field projection are defined in options that are provided via `QueryOptions` can be extracted in used in MongoDB Find calls using the `FindOptions` method:

```go
fo, err := builder.FindOptions(opt)
if err != nil {
  // this only occurs when strict schema validation is true
  // and a field is named in the querystring that doesn't actually
  // exist as defined in the schema... this is NOT the default
  // behavior
}
```

##### Projection

Projection is supported by specifying fields via the `fields` querystring parameter. By default, no projection is specified and all fields are returned. A `-` prefix can be used before a field name to exclude it from the results.

* `?fields=name,age,-someDate`: includes the fields `name`, `age`, but not `someDate`
* `?fields=-_id,someID,name`: excludes `_id`, but includes `name` and `someID`

##### Pagination

The `QueryBuilder` attempts to determine if `page` and `size` or `offset` and `limit` is used as a pagination strategy based on the `QueryOptions` values.

* `?page[limit]=100&page[offset]=0`: sets `skip` to 0 and `limit` to 100
* `?page[size]=100&page[page]=1`: sets `skip` to 100 and `limit` to 100

##### Sort

Sort is supported by specifying fields in the `sort` querystring parameter.

* `?sort=-someDate,name`: sorts descending by `someDate` and ascending by `name`
