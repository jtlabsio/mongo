// main package is an example of how to use the querybuilder to
// construct filters for MongoDB Find operations.
//
// To run this example, get a running instance of Docker on 27017
// `docker run -d --name example-mongo -p 27017:27017 mongo`
//
// To start the example server:
// `go run examples/example.go`
//
// To query the newly running example API:
// `curl http://localhost:3080/v1/things?filter[attributes]=round`
//
// For more queryoptions info, see: https://github.com/brozeph/queryoptions
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	querybuilder "github.com/brozeph/mongoquerybuilder"
	"github.com/brozeph/queryoptions"
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
			"attributes": bson.M{
				"bsonType":    "array",
				"description": "type tags for the thing",
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

	http.HandleFunc("/v1/things", getThings)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
