package mongo

import (
	"github.com/brozeph/queryoptions"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func BuildFilterAndOptions(qo queryoptions.Options) (map[string]interface{}, *options.FindOptions, error) {
	// apply pagination
	limit := qo.Page["limit"]
	offset := qo.Page["offset"]
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	// apply projection (filter[fields])
	if fields, ok := qo.Filter["fields"]; ok {
		prj := map[string]int{}
		for _, field := range fields {
			if field[0:1] == "-" {
				prj[field[1:]] = 0
				continue
			}

			prj[field] = 1
		}

		opts.SetProjection(prj)
	}

	// apply sorting
	if qo.Sort != nil && len(qo.Sort) > 0 {
		sort := map[string]int{}
		for _, field := range qo.Sort {
			switch field[0:1] {
			case "-":
				sort[field[1:]] = -1
				continue
			case "+":
				sort[field[1:]] = 1
				continue
			default:
				sort[field] = 1
			}
		}

		opts.SetSort(sort)
	}

	return nil, nil, nil
}
