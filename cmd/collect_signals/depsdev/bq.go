package depsdev

import (
	"context"
	"errors"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

var NoResultError = errors.New("no results returned")

type Dataset struct {
	ds *bigquery.Dataset
}

type Table struct {
	md *bigquery.TableMetadata
}

// bqAPI wraps the BigQuery Go API to make the deps.dev implementation easier to unit test.
type bqAPI interface {
	Project() string
	OneResultQuery(ctx context.Context, query string, params map[string]any, result any) error
	NoResultQuery(ctx context.Context, query string, params map[string]any) error
	GetDataset(ctx context.Context, id string) (*Dataset, error)
	CreateDataset(ctx context.Context, id string) (*Dataset, error)
	GetTable(ctx context.Context, d *Dataset, id string) (*Table, error)
}

type bq struct {
	client *bigquery.Client
}

func (b *bq) Project() string {
	return b.client.Project()
}

func (b *bq) OneResultQuery(ctx context.Context, query string, params map[string]any, result any) error {
	q := b.client.Query(query)
	for k, v := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{Name: k, Value: v})
	}
	it, err := q.Read(ctx)
	if err != nil {
		return err
	}
	err = it.Next(result)
	if err == iterator.Done {
		return NoResultError
	}
	if err != nil {
		return err
	}
	return nil
}

func (b *bq) NoResultQuery(ctx context.Context, query string, params map[string]any) error {
	q := b.client.Query(query)
	for k, v := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{Name: k, Value: v})
	}
	j, err := q.Run(ctx)
	if err != nil {
		return err
	}
	status, err := j.Wait(ctx)
	if err != nil {
		return err
	}
	if err := status.Err(); err != nil {
		return err
	}
	return nil
}

func (b *bq) GetDataset(ctx context.Context, id string) (*Dataset, error) {
	ds := b.client.Dataset(id)
	_, err := ds.Metadata(ctx)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Dataset{
		ds: ds,
	}, nil
}

func (b *bq) CreateDataset(ctx context.Context, id string) (*Dataset, error) {
	ds := b.client.Dataset(id)
	if err := ds.Create(ctx, &bigquery.DatasetMetadata{}); err != nil {
		return nil, err
	}
	return &Dataset{
		ds: ds,
	}, nil
}

func (b *bq) GetTable(ctx context.Context, d *Dataset, id string) (*Table, error) {
	t := d.ds.Table(id)
	md, err := t.Metadata(ctx)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Table{md: md}, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	apiErr, ok := err.(*googleapi.Error)
	return ok && apiErr.Code == 404
}
