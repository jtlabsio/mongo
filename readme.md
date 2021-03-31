# MongoDB QueryBuilder

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/brozeph/mongoquerybuilder) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/brozeph/mongoquerybuilder/main/LICENSE) [![Coverage](http://gocover.io/_badge/github.com/brozeph/mongoquerybuilder)](http://gocover.io/github.com/brozeph/mongoquerybuilder)


This library exists to ease the creation of MongoDB filter and FindOptions structs when using the MongoDB driver in combination with a [JSONAPI query parser](https://github.com/brozeph/queryoptions).

## Usage

```go
package main

import (
  "fmt"
  "log"
  "net/http"

  "github.com/brozeph/queryoptions"
  "github.com/brozeph/mongoquerybuilder"
  "go.mongodb.org/mongo-driver/bson"
  "go.mongodb.org/mongo-driver/bson/primitive"
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

  // build a bson.D filter for the Find based on queryoptions filters
  filter, err :=  builder.Filter(opt)
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

  defer cur.Close()

  data := []interface{}{}
  if err = cur.All(ctx, &data); err != nil {
    fmt.Fprint(w, err)
    return
  }

  fmt.Fprint(w, "")
}

func main() {
  // create a MongoDB client
  mc, err := mongo.NewClient(options.Client().ApplyURI("mongo://localhost:27017"))
  if err != nil {
    log.Fatal(err)
  }

  // connect to MongoDB
  if err := mc.Connect(context.TODO()); err != nil {
    log.Fatal(err)
  }

  // create a collection with the schema
  colOpts := options.CreateCollection().SetValidator(thingsSchema)
  if err := mc.Database("things-db").CreateCollection(context.TODO(), "things", colOpts); err != nil {
    log.Fatal(err)
  }

  // set the collection pointer
  collection = mc.Database("things-db").Collection("things")

  http.HandleFunc("/v1/things", getThings)
  log.Fatal(http.ListenAndServe(":8080", nil))
}
```