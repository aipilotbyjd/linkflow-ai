// Package nodes provides S3/Storage node implementation
package nodes

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&S3Node{})
}

// S3Node implements S3/Storage operations
type S3Node struct{}

func (n *S3Node) GetType() string { return "s3" }
func (n *S3Node) Validate(config map[string]interface{}) error { return nil }

func (n *S3Node) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "s3",
		Name:        "AWS S3",
		Description: "Upload, download, and manage files in AWS S3",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "aws-s3",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Upload", Value: "upload"}, {Label: "Download", Value: "download"},
				{Label: "Delete", Value: "delete"}, {Label: "List", Value: "list"},
				{Label: "Copy", Value: "copy"}, {Label: "Get Presigned URL", Value: "getPresignedUrl"},
			}},
			{Name: "bucket", Type: "string", Required: true},
			{Name: "key", Type: "string"},
			{Name: "content", Type: "string"},
			{Name: "contentType", Type: "string"},
			{Name: "prefix", Type: "string"},
			{Name: "expiresIn", Type: "number", Default: 3600},
		},
	}
}

func (n *S3Node) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	accessKeyID, _ := input.Credentials["accessKeyId"].(string)
	secretAccessKey, _ := input.Credentials["secretAccessKey"].(string)
	region, _ := input.Credentials["region"].(string)
	endpoint, _ := input.Credentials["endpoint"].(string)

	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	operation, _ := input.NodeConfig["operation"].(string)
	bucket, _ := input.NodeConfig["bucket"].(string)
	key, _ := input.NodeConfig["key"].(string)

	var result map[string]interface{}

	switch operation {
	case "upload":
		result, err = n.upload(ctx, client, input.NodeConfig)
	case "download":
		result, err = n.download(ctx, client, bucket, key, input.NodeConfig)
	case "delete":
		result, err = n.deleteObject(ctx, client, bucket, key)
	case "copy":
		result, err = n.copyObject(ctx, client, input.NodeConfig)
	case "move":
		result, err = n.moveObject(ctx, client, input.NodeConfig)
	case "list":
		result, err = n.listObjects(ctx, client, bucket, input.NodeConfig)
	case "getMetadata":
		result, err = n.getMetadata(ctx, client, bucket, key)
	case "createBucket":
		result, err = n.createBucket(ctx, client, bucket, region)
	case "deleteBucket":
		result, err = n.deleteBucket(ctx, client, bucket)
	case "listBuckets":
		result, err = n.listBuckets(ctx, client)
	case "getPresignedUrl":
		result, err = n.getPresignedUrl(ctx, client, bucket, key, input.NodeConfig)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *S3Node) upload(ctx context.Context, client *s3.Client, config map[string]interface{}) (map[string]interface{}, error) {
	bucket, _ := config["bucket"].(string)
	key, _ := config["key"].(string)
	content, _ := config["content"].(string)
	contentType, _ := config["contentType"].(string)
	encoding, _ := config["encoding"].(string)
	acl, _ := config["acl"].(string)

	var body []byte
	switch encoding {
	case "base64":
		var err error
		body, err = base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, fmt.Errorf("invalid base64: %w", err)
		}
	default:
		body = []byte(content)
	}

	if contentType == "" {
		contentType = http.DetectContentType(body)
		if contentType == "application/octet-stream" {
			// Try to guess from extension
			ext := filepath.Ext(key)
			switch ext {
			case ".json":
				contentType = "application/json"
			case ".txt":
				contentType = "text/plain"
			case ".html":
				contentType = "text/html"
			case ".css":
				contentType = "text/css"
			case ".js":
				contentType = "application/javascript"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".gif":
				contentType = "image/gif"
			case ".pdf":
				contentType = "application/pdf"
			}
		}
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	}

	if acl != "" {
		input.ACL = s3Types.ObjectCannedACL(acl)
	}

	result, err := client.PutObject(ctx, input)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":      bucket,
		"key":         key,
		"etag":        aws.ToString(result.ETag),
		"contentType": contentType,
		"size":        len(body),
	}, nil
}

func (n *S3Node) download(ctx context.Context, client *s3.Client, bucket, key string, config map[string]interface{}) (map[string]interface{}, error) {
	encoding, _ := config["encoding"].(string)

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	var content string
	switch encoding {
	case "base64":
		content = base64.StdEncoding.EncodeToString(body)
	default:
		content = string(body)
	}

	return map[string]interface{}{
		"bucket":       bucket,
		"key":          key,
		"content":      content,
		"contentType":  aws.ToString(result.ContentType),
		"size":         len(body),
		"lastModified": result.LastModified,
		"etag":         aws.ToString(result.ETag),
	}, nil
}

func (n *S3Node) deleteObject(ctx context.Context, client *s3.Client, bucket, key string) (map[string]interface{}, error) {
	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":  bucket,
		"key":     key,
		"deleted": true,
	}, nil
}

func (n *S3Node) copyObject(ctx context.Context, client *s3.Client, config map[string]interface{}) (map[string]interface{}, error) {
	sourceBucket, _ := config["sourceBucket"].(string)
	sourceKey, _ := config["sourceKey"].(string)
	destBucket, _ := config["destinationBucket"].(string)
	destKey, _ := config["destinationKey"].(string)

	if destBucket == "" {
		destBucket = sourceBucket
	}

	copySource := fmt.Sprintf("%s/%s", sourceBucket, sourceKey)

	result, err := client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(destBucket),
		Key:        aws.String(destKey),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"sourceBucket":      sourceBucket,
		"sourceKey":         sourceKey,
		"destinationBucket": destBucket,
		"destinationKey":    destKey,
		"etag":              aws.ToString(result.CopyObjectResult.ETag),
	}, nil
}

func (n *S3Node) moveObject(ctx context.Context, client *s3.Client, config map[string]interface{}) (map[string]interface{}, error) {
	// Copy first
	result, err := n.copyObject(ctx, client, config)
	if err != nil {
		return nil, err
	}

	// Then delete source
	sourceBucket, _ := config["sourceBucket"].(string)
	sourceKey, _ := config["sourceKey"].(string)
	_, err = n.deleteObject(ctx, client, sourceBucket, sourceKey)
	if err != nil {
		return nil, fmt.Errorf("copy succeeded but delete failed: %w", err)
	}

	result["moved"] = true
	return result, nil
}

func (n *S3Node) listObjects(ctx context.Context, client *s3.Client, bucket string, config map[string]interface{}) (map[string]interface{}, error) {
	prefix, _ := config["prefix"].(string)
	maxKeys := int32(1000)
	if mk, ok := config["maxKeys"].(float64); ok {
		maxKeys = int32(mk)
	}

	result, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(maxKeys),
	})
	if err != nil {
		return nil, err
	}

	objects := make([]map[string]interface{}, len(result.Contents))
	for i, obj := range result.Contents {
		objects[i] = map[string]interface{}{
			"key":          aws.ToString(obj.Key),
			"size":         obj.Size,
			"lastModified": obj.LastModified,
			"etag":         aws.ToString(obj.ETag),
			"storageClass": string(obj.StorageClass),
		}
	}

	return map[string]interface{}{
		"bucket":      bucket,
		"prefix":      prefix,
		"objects":     objects,
		"count":       len(objects),
		"isTruncated": result.IsTruncated,
	}, nil
}

func (n *S3Node) getMetadata(ctx context.Context, client *s3.Client, bucket, key string) (map[string]interface{}, error) {
	result, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":       bucket,
		"key":          key,
		"contentType":  aws.ToString(result.ContentType),
		"size":         aws.ToInt64(result.ContentLength),
		"lastModified": result.LastModified,
		"etag":         aws.ToString(result.ETag),
		"metadata":     result.Metadata,
	}, nil
}

func (n *S3Node) createBucket(ctx context.Context, client *s3.Client, bucket, region string) (map[string]interface{}, error) {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// LocationConstraint is required for non-us-east-1 regions
	if region != "us-east-1" {
		input.CreateBucketConfiguration = &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(region),
		}
	}

	_, err := client.CreateBucket(ctx, input)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":  bucket,
		"created": true,
	}, nil
}

func (n *S3Node) deleteBucket(ctx context.Context, client *s3.Client, bucket string) (map[string]interface{}, error) {
	_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":  bucket,
		"deleted": true,
	}, nil
}

func (n *S3Node) listBuckets(ctx context.Context, client *s3.Client) (map[string]interface{}, error) {
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	buckets := make([]map[string]interface{}, len(result.Buckets))
	for i, b := range result.Buckets {
		buckets[i] = map[string]interface{}{
			"name":         aws.ToString(b.Name),
			"creationDate": b.CreationDate,
		}
	}

	return map[string]interface{}{
		"buckets": buckets,
		"count":   len(buckets),
	}, nil
}

func (n *S3Node) getPresignedUrl(ctx context.Context, client *s3.Client, bucket, key string, config map[string]interface{}) (map[string]interface{}, error) {
	expiresIn := int64(3600)
	if exp, ok := config["expiresIn"].(float64); ok {
		expiresIn = int64(exp)
	}

	presignClient := s3.NewPresignClient(client)
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiresIn) * time.Second
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":    bucket,
		"key":       key,
		"url":       result.URL,
		"expiresIn": expiresIn,
	}, nil
}


