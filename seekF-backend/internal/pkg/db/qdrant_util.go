package db

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
)

var qdrantClient *QdrantUtil

type QdrantUtil struct {
	client *qdrant.Client
}

func InitQdrant() error {
	cfg := configs.GetConfig()

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: cfg.QdrantConfig.Host,
		Port: cfg.QdrantConfig.Port,
	})
	if err != nil {
		return err
	}

	qdrantClient = &QdrantUtil{client: client}
	zlog.Info(fmt.Sprintf("connected to qdrant at %s:%d", cfg.QdrantConfig.Host, cfg.QdrantConfig.Port))
	return nil
}

func GetQdrant() *QdrantUtil {
	return qdrantClient
}

func (q *QdrantUtil) EnsureCollection(ctx context.Context, collectionName string, vectorSize uint64) error {
	exists, err := q.client.CollectionExists(ctx, collectionName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig:  qdrant.NewVectorsConfig(&qdrant.VectorParams{Size: vectorSize, Distance: qdrant.Distance_Cosine}),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("created qdrant collection: %s", collectionName))
	return nil
}

func (q *QdrantUtil) DeleteCollection(ctx context.Context, collectionName string) error {
	err := q.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("deleted qdrant collection: %s", collectionName))
	return nil
}

func (q *QdrantUtil) UpsertChunks(ctx context.Context, collectionName string, chunks []string, vectors [][]float32, docUUID string) error {
	points := make([]*qdrant.PointStruct, len(chunks))
	for i, chunk := range chunks {
		points[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(i)),
			Vectors: qdrant.NewVectors(vectors[i]...),
			Payload: qdrant.NewValueMap(map[string]any{
				"text":     chunk,
				"doc_uuid": docUUID,
			}),
		}
	}

	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("upserted %d chunks to collection %s", len(chunks), collectionName))
	return nil
}

func (q *QdrantUtil) DeleteByDocUUID(ctx context.Context, collectionName string, docUUID string) error {
	filter := &qdrant.Filter{
		Should: []*qdrant.Condition{
			qdrant.NewMatchKeyword("doc_uuid", docUUID),
		},
	}

	wait := true
	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Wait:           &wait,
		Points:         qdrant.NewPointsSelectorFilter(filter),
	})
	if err != nil {
		return err
	}

	zlog.Info(fmt.Sprintf("deleted chunks for doc_uuid %s from collection %s", docUUID, collectionName))
	return nil
}

func (q *QdrantUtil) Search(ctx context.Context, collectionName string, queryVector []float32, topK int) ([]string, error) {
	limit := uint64(topK)
	result, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQuery(queryVector...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, err
	}

	var results []string
	for _, point := range result {
		if point.Payload != nil {
			if text, ok := point.Payload["text"]; ok {
				textStr := text.GetStringValue()
				if textStr != "" {
					results = append(results, textStr)
				}
			}
		}
	}

	return results, nil
}

func (q *QdrantUtil) Close() error {
	return q.client.Close()
}
