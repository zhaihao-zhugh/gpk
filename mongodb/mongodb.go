package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// type CollectionInterface struct {
// 	col   *mongo.Collection
// 	model interface{}
// }

func (cfg *MongoConfig) address() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.Username, cfg.Password, cfg.Host, cfg.Port)
}

var c = new(mongo.Database)

func NewClient(cfg *MongoConfig, database string) {
	cli, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.address()))
	if err != nil {
		panic(err)
	}
	c = cli.Database(database)
}

func NewCollection(collection string) *mongo.Collection {
	return c.Collection(collection)
}

func Create(collection string, data []interface{}) (interface{}, error) {
	col := c.Collection(collection)
	r, err := col.InsertMany(context.Background(), data)
	if err != nil {
		return nil, err
	}
	return r.InsertedIDs, nil
}

func Delete(collection string, filter map[string]interface{}) (interface{}, error) {
	col := c.Collection(collection)
	res, err := col.DeleteMany(context.Background(), map2bsonD(filter))
	if err != nil {
		return nil, err
	}
	return res.DeletedCount, nil
}

func UpdateByID(collection string, id string, data map[string]interface{}) (interface{}, error) {
	col := c.Collection(collection)
	_id, _ := primitive.ObjectIDFromHex(id)
	_, err := col.UpdateByID(context.Background(), _id, bson.D{{Key: "$set", Value: map2bsonD(data)}})
	if err != nil {
		return nil, err
	}
	return id, nil
}

func UpdateOne(collection string, filter map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	col := c.Collection(collection)
	res, err := col.UpdateOne(context.Background(), map2bsonD(filter), bson.D{{Key: "$set", Value: map2bsonD(data)}})
	if err != nil {
		return nil, err
	}
	return res.ModifiedCount, nil
}

func UpdateMany(collection string, filter map[string]interface{}, data map[string]interface{}) (interface{}, error) {
	col := c.Collection(collection)
	res, err := col.UpdateMany(context.Background(), map2bsonD(filter), bson.D{{Key: "$set", Value: map2bsonD(data)}})
	if err != nil {
		return nil, err
	}
	return res.ModifiedCount, nil
}

func Find(collection string, params map[string]interface{}) (total int64, results []map[string]interface{}, err error) {
	col := c.Collection(collection)
	filter, opts := handleParams(params)
	fmt.Println("filter ", filter)
	total, err = col.CountDocuments(context.TODO(), filter)
	if err != nil {
		return
	}
	res, err := col.Find(context.Background(), filter, opts)
	if err != nil {
		return
	}
	if err = res.All(context.TODO(), &results); err != nil {
		return
	}
	for _, v := range results {
		v["id"] = v["_id"]
		delete(v, "_id")
	}
	return
}

func Agg(collection string, data ...map[string]interface{}) (results []map[string]interface{}, err error) {
	col := c.Collection(collection)
	opts := options.Aggregate().SetMaxTime(1 * time.Minute)
	var p mongo.Pipeline
	for _, v := range data {
		p = append(p, map2bsonD(v))
	}
	res, err := col.Aggregate(context.Background(), p, opts)
	if err != nil {
		return
	}
	err = res.All(context.TODO(), &results)
	return
}

func CreateIndex(collection string, n_index map[string]interface{}) error {
	indexModel := mongo.IndexModel{
		Keys: map2bsonD(n_index),
	}
	col := c.Collection(collection)
	name, err := col.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		return err
	}
	fmt.Println("Name of Index Created: " + name)
	return err
}

func handleParams(params map[string]interface{}) (result bson.D, opts *options.FindOptions) {
	result = bson.D{}
	opts = new(options.FindOptions)
	if params == nil {
		return
	}
	for k, v := range params {
		switch k {
		case "limit":
			if vv, ok := v.(int64); ok {
				opts.SetLimit(vv)
			}
		case "offset":
			if vv, ok := v.(int64); ok {
				opts.SetSkip(vv)
			}
		case "ordering":
			if vv, ok := v.(string); ok {
				sort := bson.D{}
				values := strings.Split(vv, ",")
				for _, value := range values {
					if strings.HasPrefix(value, "-") {
						sort = append(sort, bson.E{Key: value[1:], Value: -1})
					} else {
						sort = append(sort, bson.E{Key: value, Value: 1})
					}
				}
				opts.SetSort(sort)
				opts.SetAllowDiskUse(true)
			}
		case "fields":
			if vv, ok := v.(string); ok {
				projection := bson.D{}
				values := strings.Split(vv, ",")
				for _, value := range values {
					projection = append(projection, bson.E{Key: value, Value: 1})
				}
				opts.SetProjection(projection)
			}
		default:
			i := bson.E{Key: k, Value: v}
			if k == "id" {
				i.Key = "_id"
				i.Value, _ = primitive.ObjectIDFromHex(v.(string))
			}
			result = append(result, i)
		}
	}
	return
}

func map2bsonD(data map[string]interface{}) (result bson.D) {
	for k, v := range data {
		i := bson.E{Key: k, Value: v}
		if k == "id" {
			i.Key = "_id"
			i.Value, _ = primitive.ObjectIDFromHex(v.(string))
		}
		result = append(result, i)
	}
	return
}

func Str2ObjectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}
