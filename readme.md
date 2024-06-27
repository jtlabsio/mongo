# MongoDB QueryBuilder

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/go.jtlabs.io/mongo) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/brozeph/mongoquerybuilder/main/LICENSE) [![Coverage](https://github.com/jtlabsio/mongo/actions/workflows/ci.yml)](https://github.com/jtlabsio/mongo/actions/workflows/ci.yml/badge.svg) [![GoReportCard example](https://goreportcard.com/badge/github.com/jtlabsio/mongo)](https://goreportcard.com/report/github.com/jtlabsio/mongo)

This library exists to ease the creation of MongoDB filter and FindOptions structs when using the MongoDB driver in combination with a [JSONAPI query parser](https://github.com/jtlabsio/query).

## Contents

- [Installation](#installation)
- [Usage](#usage)
  - [QueryBuilder](#querybuilder)
    - [NewQueryBuilder](#newquerybuilder)
    - [Filter](#filter)
      - [Query Operators](#query-operators)
      - [Logical Operators](#logical-operators)
    - [FindOptions](#findoptions)
      - [Projection](#projection)
      - [Pagination](#pagination)
      - [Sort](#sort)
  - [UpdateBuilder](#updatebuilder)
    - [NewUpdateBuilder](#newupdatebuilder)
    - [UpdateOptions](#updateoptions)

## Installation

```bash
go get -u go.jtlabs.io/mongo
```

## Usage

Example code below translated from [examples/example.go](examples/example.go) - for more info, see the example file run running instructions.

```go
// main package is an example of how to use the querybuilder to
// construct filters for MongoDB Find operations. Additionally, this
// example demonstrates how to use the updatebuilder to construct
// update documents for MongoDB Update operations.
//
// To run this example, get a running instance of Docker on 27017
// `docker run -d --name example-mongo -p 27017:27017 mongo`
//
// To start the example server:
// `go run examples/example.go`
//
// To query the newly running example API:
// `curl http://localhost:8080/v1/things?filter[attributes]=round`
//
// To update a thing:
// `curl -X PUT -d '{"thingID":"123455","attributes":["expensive"]}' http://localhost:8080/v1/things`
//
// For more queryoptions info, see: https://github.com/jtlabsio/query
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
  "required": bson.A{"thingID"},
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
   "attributes": bson.M{
    "bsonType":    "array",
    "description": "type tags for the thing",
    "items": bson.M{
     "bsonType": "string",
    },
   },
  },
 },
}

// golang type for the things...
type thing struct {
 ThingID    string    `bson:"thingID"`
 Name       string    `bson:"name"`
 Created    time.Time `bson:"created"`
 Attributes []string  `bson:"attributes"`
}

// create a new MongoDB QueryBuilder (with strict validation set to true)
var queryBuilder = mongobuilder.NewQueryBuilder("things", thingsSchema, true)

// create a new MongoDB UpdateBuilder
var updateBuilder = mongobuilder.NewUpdateBuilder(
 "things",
 thingsSchema,
 mongobuilder.UpdateOptions().SetAddToSet("attributes", true),
 mongobuilder.UpdateOptions().SetIgnoreFields("thingID"),
)

// pointer for the mongo collection to query from
var collection *mongo.Collection

func getOrSetThings(w http.ResponseWriter, r *http.Request) {
 // update mongo if the requst is a PUT
 if r.Method == http.MethodPut {
  // parse the request body into a thing struct
  var t thing
  if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
   fmt.Fprint(w, err)
   return
  }

  // create an update document
  ud, err := updateBuilder.Update(t)
  if err != nil {
   fmt.Fprint(w, err)
   return
  }

  // update the mongo collection
  if _, err := collection.UpdateOne(
   context.TODO(),
   bson.M{"thingID": t.ThingID},
   ud,
   &options.UpdateOptions{
    Upsert: &[]bool{true}[0],
   }); err != nil {
   fmt.Fprint(w, err)
   return
  }

  fmt.Fprint(w, "updated")
  return
 }

 opt, err := queryoptions.FromQuerystring(r.URL.RawQuery)
 if err != nil {
  fmt.Fprint(w, err)
  return
 }

 // build a bson.M filter for the Find based on queryoptions filters
 filter, err := queryBuilder.Filter(opt)
 if err != nil {
  // NOTE: will only error when strictValidation is true
  fmt.Fprint(w, err)
  return
 }

 // build options (pagination, sorting, field projection) based on queryoptions
 fo, err := queryBuilder.FindOptions(opt)
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

 data := []thing{}
 if err = cur.All(context.TODO(), &data); err != nil {
  fmt.Fprint(w, err)
  return
 }

 re, _ := json.Marshal(data)
 fmt.Fprint(w, string(re))
}

func main() {
 // create a MongoDB client
 mc, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
 if err != nil {
  log.Fatal(err)
 }

 // create a collection with the schema
 colOpts := options.CreateCollection().SetValidator(thingsSchema)
 if err := mc.Database("things-db").CreateCollection(context.TODO(), "things", colOpts); err == nil {
  // if err is nil, this is the first time the program is running... insert data
  // I know... kinda whack, but this is just an example
  data := []interface{}{
   thing{
    ThingID:    "123456",
    Name:       "basketball",
    Created:    time.Now(),
    Attributes: []string{"round", "orange", "bouncey"},
   },
   thing{
    ThingID:    "123455",
    Name:       "computer",
    Created:    time.Now(),
    Attributes: []string{"square", "metal", "heavy"},
   },
   thing{
    ThingID:    "123454",
    Name:       "superball",
    Created:    time.Now(),
    Attributes: []string{"round", "bouncey", "small"},
   },
   thing{
    ThingID:    "123453",
    Name:       "glass",
    Created:    time.Now(),
    Attributes: []string{"glass", "container", "transparent"},
   },
   thing{
    ThingID:    "123452",
    Name:       "can",
    Created:    time.Now(),
    Attributes: []string{"metal", "cylinder", "empty"},
   },
  }
  mc.Database("things-db").Collection("things").InsertMany(context.TODO(), data)
 }

 // set the collection pointer
 collection = mc.Database("things-db").Collection("things")

 http.HandleFunc("/v1/things", getOrSetThings)
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
jsonSchema := bson.M{ /* a JSON schema here... */ }
qb := querybuilder.NewQueryBuilder("collectionName", jsonSchema, true)
```

##### Schemas

The schema can be provided as a `bson.M`, `map[string]any`, `string`, or a `[]byte` and should be a valid JSON schema that is used to validate the collection. The schema is used to coerce types and validate the fields that are provided in the querystring.

```go
jsonSchema := map[string]any{
 "$jsonSchema": map[string]any{
  "bsonType": "object",
  "required": []string{"thingID"},
  "properties": map[string]any{
   "thingID": map[string]any{
    "bsonType":    "string",
    "description": "primary identifier for the thing",
   },
   "created": map[string]any{
    "bsonType":    "date",
    "description": "time at which the thing was created",
   },
   "name": map[string]any{
    "bsonType":    "string",
    "description": "name of the thing",
   },
   "types": map[string]any{
    "bsonType":    "array",
    "description": "type tags for the thing",
    "items": map[string]any{
     "bsonType": "string",
    },
   },
  },
 },
}

// create a new MongoDB QueryBuilder
var builder = querybuilder.NewQueryBuilder("things", thingsSchema)
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

##### Query Options

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

####### string bsonType

For `string` bsonType fields in the schema, the following operators can be leveraged with specific querystring hints:

- `begins with` (i.e. `{ "name": { "regex": /^term/, "options": "i" } }`): `?filter[name]=term*`
- `ends with` (i.e. `{ "name": { "regex": /term$/, "options": "i" } }`): `?ilter[name]=*term`
- `exact match` (i.e. `{ "name": { "regex": /^term$/ } }`): `?filter[name]="term"`
- `not equal` (i.e. `{ "name": { "$ne": "term" } }`): `?filter[name]=!=term`
- `in` (i.e. `{ "name": { "$in": [ ... ] } }`): `?filter[name]=term1,term2,term3,term4`
- standard comparison (i.e. `{ "name": "term" }`): `?filter[name]=term`
- `null` is translated to `null` in the query (i.e. `{ 'name': null }`): `?filter[name]=null`

####### numeric bsonType

For `numeric` bsonType fields in the schema (`int`, `long`, `decimal`, and `double`), any values provided in the querystring that are parsed by `QueryOptions` are coerced to the appropriate type when constructing the filter. Additionally, the following operators can be used in combination with querystring hints:

- `less than` (i.e. `{ "age": { "$lt": 5 } }`): `?filter[age]=<5`
- `less than equal` (i.e. `{ "age": { "$lte": 5 } }`): `?filter[age]=<=5`
- `greater than` (i.e. `{ "age": { "$gt": 5 } }`): `?filter[age]=>5`
- `greater than equal` (i.e. `{ "age": { "$gte": 5 } }`): `?filter[age]=>=5`
- `not equals` (i.e. `{ "age": { "$ne": 5 } }`): `?filter[age]=!=5`
- `in` (i.e. `{ "age": { "$in": [1,2,3,4,5] } }`): `?filter[age]=1,2,3,4,5`
- standard comparison (i.e. `{ "age": 5 }`): `?filter[age]=5`

####### date bsonType

For `date` bsonType fields in the schema (`date` and `timestamp`), any values in the querystring are converted according to `RFC3339` and used in the filter. The following operators can be used in combination with querystring hints:

- `less than` (i.e. `{ "someDate": { "$lt": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=<2021-02-16T02:04:05.000Z`
- `less than equal` (i.e. `{ "someDate": { "$lte": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=<=2021-02-16T02:04:05.000Z`
- `greater than` (i.e. `{ "someDate": { "$gt": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=>2021-02-16T02:04:05.000Z`
- `greater than equal` (i.e. `{ "someDate": { "$gte": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=>=2021-02-16T02:04:05.000Z`
- `not equals` (i.e. `{ "someDate": { "$ne": new Date("2021-02-16T02:04:05.000Z") } }`): `?filter[someDate]=!=2021-02-16T02:04:05.000Z`
- `in` (i.e. `{ "someDate": { "$in": [ ... ] } }`): `?filter[someDate]=2021-02-16T00:00:00.000Z,2021-02-15T00:00:00.000Z`
- standard comparison (i.e. `{ "someDate": new Date("2021-02-16T02:04:05.000Z") }`): `?filter[someDate]=2021-02-16T02:04:05.000Z`

###### Logical Operators

By default, when one or more query operators are provided via the search querystring, the `QueryBuilder` will construct a `$and` filter with the provided operators. For example:

- `greater than equal` combined with `less than` (i.e. `{ $"and": [ { "age": { "$gte": 18, } }, { "age": { "$lt": 24, } } ] }`): `?filter[age]=>=18,<24`

This behavior can be overridden by providing a LogicalOperator constant as an optional value to the `Filter` method:

```go
// a query filter in a bson.M based on QueryOptions Filter values
f, err := builder.Filter(opt, mongobuilder.Or)
if err != nil {
  // this only occurs when strict schema validation is true
  // and a field is named in the querystring that doesn't actually
  // exist as defined in the schema... this is NOT the default
  // behavior
}
```

There are 4 logical operators that can be used:

- `mongobuilder.And`: `$and`
- `mongobuilder.Or`: `$or`
- `mongobuilder.Nor`: `$nor`
- `mongobuilder.Not`: `$not`

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

- `?fields=name,age,-someDate`: includes the fields `name`, `age`, but not `someDate`
- `?fields=-_id,someID,name`: excludes `_id`, but includes `name` and `someID`

##### Pagination

The `QueryBuilder` attempts to determine if `page` and `size` or `offset` and `limit` is used as a pagination strategy based on the `QueryOptions` values.

- `?page[limit]=100&page[offset]=0`: sets `skip` to 0 and `limit` to 100
- `?page[size]=100&page[page]=1`: sets `skip` to 100 and `limit` to 100

##### Sort

Sort is supported by specifying fields in the `sort` querystring parameter.

- `?sort=-someDate,name`: sorts descending by `someDate` and ascending by `name`

### UpdateBuilder

The `UpdateBuilder` struct can be used to create update operations for MongoDB collections. The results of `UpdateBuilder` can be used when calling any MongoDB driver update operations, including `FindOneAndUpdate`, `UpdateOne` and `UpdateMany`, etc.

#### NewUpdateBuilder

New `UpdateBuilder` instances can be created using the `NewUpdateBuilder` function:

```go
func example() {
  schema := `{
    "$jsonSchema": {
      "bsonType": "object",
      "properties": {
        "tagList": {
          "bsonType": "array",
          "description": "list of tags",
          "items": {
            "bsonType": "string"
          }
        },
        "authorList": {
        "bsonType": "array",
        "description": "list of authors",
        "items": {
          "bsonType": "object",
            "properties": {
              "name": {
                "bsonType": "string",
                "description": "name of the author"
              },
              "email": {
                "bsonType": "string",
                "description": "email of the author"
              }
            }
          }
        }
      }
    }
  }`
  
  // create an update builder
  ub := NewUpdateBuilder("collection", schema)
  
  // create an update document that uses $addToSet for tagList (but use $set for authorList)
  opts := UpdateOptions().SetAddToSet("tagList", true)
  
  // retrieve the update document
  doc := article{
    TagList: []string{"new", "tag"},
    AuthorList: []author{
      {
        Name:  "John Doe",
        Email: "joh@n.do.e",
      },
    },
  }

  // now do something with the update document...
  // which looks something like this:
  // bson.D{
  //  {"$addToSet", bson.D{
  //    {"tagList", bson.D{
  //      {"$each", []string{"new", "tag"}},
  //    }},
  //  }},
  //  {"$set", bson.D{
  //    {"authorList", bson.A[]{
  //      bson.D{
  //        {"name", "John Doe"},
  //        {"email", "joh@n.do.e",
  //      },
  //    }},
  //  }},
  // }
  update, err := ub.Update(doc, opts)
  if err != nil {
    fmt.Println(err)
    return
  }
}
```

#### UpdateOptions

The `UpdateOptions` struct can be used to specify the type of update operation that should be performed on a field in the document. The following methods are available:

- `SetAddToSet`: sets the update operation to `$addToSet` for the specified field
- `SetIgnoreFields`: sets the fields that should be ignored when constructing the update document
- `SetStrictValidation`: sets the strict validation flag for the update builder
- `SetUnsetWhenEmpty`: sets the flag to unset fields when they are empty in the document

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
