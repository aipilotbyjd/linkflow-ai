// Package nodes provides MongoDB node implementation
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	runtime.Register(&MongoDBNode{})
}

// MongoDBNode implements MongoDB operations
type MongoDBNode struct{}

func (n *MongoDBNode) GetType() string { return "mongodb" }
func (n *MongoDBNode) Validate(config map[string]interface{}) error { return nil }

func (n *MongoDBNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "mongodb",
		Name:        "MongoDB",
		Description: "Execute operations on MongoDB database",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "mongodb",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Find", Value: "find"}, {Label: "Find One", Value: "findOne"},
				{Label: "Insert One", Value: "insertOne"}, {Label: "Update One", Value: "updateOne"},
				{Label: "Delete One", Value: "deleteOne"}, {Label: "Aggregate", Value: "aggregate"},
			}},
			{Name: "collection", Type: "string", Required: true},
			{Name: "filter", Type: "json"},
			{Name: "document", Type: "json"},
			{Name: "update", Type: "json"},
			{Name: "pipeline", Type: "json"},
			{Name: "limit", Type: "number"},
		},
	}
}

func (n *MongoDBNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	connectionString, _ := input.Credentials["connectionString"].(string)
	if connectionString == "" {
		host, _ := input.Credentials["host"].(string)
		port, _ := input.Credentials["port"].(string)
		user, _ := input.Credentials["user"].(string)
		password, _ := input.Credentials["password"].(string)
		database, _ := input.Credentials["database"].(string)

		if port == "" {
			port = "27017"
		}
		if user != "" && password != "" {
			connectionString = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, password, host, port, database)
		} else {
			connectionString = fmt.Sprintf("mongodb://%s:%s/%s", host, port, database)
		}
	}

	clientOpts := options.Client().ApplyURI(connectionString)
	clientOpts.SetConnectTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Disconnect(ctx)

	database, _ := input.Credentials["database"].(string)
	collection, _ := input.NodeConfig["collection"].(string)
	coll := client.Database(database).Collection(collection)

	operation, _ := input.NodeConfig["operation"].(string)
	var result map[string]interface{}

	switch operation {
	case "find":
		result, err = n.find(ctx, coll, input.NodeConfig)
	case "findOne":
		result, err = n.findOne(ctx, coll, input.NodeConfig)
	case "insertOne":
		result, err = n.insertOne(ctx, coll, input.NodeConfig)
	case "insertMany":
		result, err = n.insertMany(ctx, coll, input.NodeConfig)
	case "updateOne":
		result, err = n.updateOne(ctx, coll, input.NodeConfig)
	case "updateMany":
		result, err = n.updateMany(ctx, coll, input.NodeConfig)
	case "deleteOne":
		result, err = n.deleteOne(ctx, coll, input.NodeConfig)
	case "deleteMany":
		result, err = n.deleteMany(ctx, coll, input.NodeConfig)
	case "aggregate":
		result, err = n.aggregate(ctx, coll, input.NodeConfig)
	case "count":
		result, err = n.count(ctx, coll, input.NodeConfig)
	case "distinct":
		result, err = n.distinct(ctx, coll, input.NodeConfig)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *MongoDBNode) find(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])
	opts := options.Find()

	if sort, ok := config["sort"].(map[string]interface{}); ok {
		opts.SetSort(n.toBSON(sort))
	}
	if projection, ok := config["projection"].(map[string]interface{}); ok {
		opts.SetProjection(n.toBSON(projection))
	}
	if limit, ok := config["limit"].(float64); ok {
		opts.SetLimit(int64(limit))
	}
	if skip, ok := config["skip"].(float64); ok {
		opts.SetSkip(int64(skip))
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"documents": results,
		"count":     len(results),
	}, nil
}

func (n *MongoDBNode) findOne(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])
	opts := options.FindOne()

	if projection, ok := config["projection"].(map[string]interface{}); ok {
		opts.SetProjection(n.toBSON(projection))
	}

	var result map[string]interface{}
	err := coll.FindOne(ctx, filter, opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return map[string]interface{}{"document": nil}, nil
	}
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{"document": result}, nil
}

func (n *MongoDBNode) insertOne(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	document := n.toBSON(config["document"])

	result, err := coll.InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"insertedId": result.InsertedID,
	}, nil
}

func (n *MongoDBNode) insertMany(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	docs, _ := config["documents"].([]interface{})
	documents := make([]interface{}, len(docs))
	for i, doc := range docs {
		documents[i] = n.toBSON(doc)
	}

	result, err := coll.InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"insertedIds":   result.InsertedIDs,
		"insertedCount": len(result.InsertedIDs),
	}, nil
}

func (n *MongoDBNode) updateOne(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])
	update := n.toBSON(config["update"])
	opts := options.Update()

	if upsert, ok := config["upsert"].(bool); ok {
		opts.SetUpsert(upsert)
	}

	result, err := coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
	}
	if result.UpsertedID != nil {
		res["upsertedId"] = result.UpsertedID
	}
	return res, nil
}

func (n *MongoDBNode) updateMany(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])
	update := n.toBSON(config["update"])
	opts := options.Update()

	if upsert, ok := config["upsert"].(bool); ok {
		opts.SetUpsert(upsert)
	}

	result, err := coll.UpdateMany(ctx, filter, update, opts)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
	}, nil
}

func (n *MongoDBNode) deleteOne(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"deletedCount": result.DeletedCount,
	}, nil
}

func (n *MongoDBNode) deleteMany(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])

	result, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"deletedCount": result.DeletedCount,
	}, nil
}

func (n *MongoDBNode) aggregate(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	pipeline, _ := config["pipeline"].([]interface{})
	stages := make([]bson.M, len(pipeline))
	for i, stage := range pipeline {
		stages[i] = n.toBSON(stage).(bson.M)
	}

	cursor, err := coll.Aggregate(ctx, stages)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"documents": results,
		"count":     len(results),
	}, nil
}

func (n *MongoDBNode) count(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	filter := n.toBSON(config["filter"])

	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"count": count,
	}, nil
}

func (n *MongoDBNode) distinct(ctx context.Context, coll *mongo.Collection, config map[string]interface{}) (map[string]interface{}, error) {
	field, _ := config["field"].(string)
	filter := n.toBSON(config["filter"])

	values, err := coll.Distinct(ctx, field, filter)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"values": values,
		"count":  len(values),
	}, nil
}

func (n *MongoDBNode) toBSON(v interface{}) interface{} {
	if v == nil {
		return bson.M{}
	}

	switch val := v.(type) {
	case map[string]interface{}:
		result := bson.M{}
		for k, v := range val {
			// Handle special MongoDB operators
			if k == "_id" {
				if idStr, ok := v.(string); ok {
					if oid, err := primitive.ObjectIDFromHex(idStr); err == nil {
						result[k] = oid
						continue
					}
				}
			}
			result[k] = n.toBSON(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = n.toBSON(item)
		}
		return result
	case string:
		// Try to convert JSON strings to BSON
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(val), &jsonVal); err == nil {
			return n.toBSON(jsonVal)
		}
		return val
	default:
		return val
	}
}
