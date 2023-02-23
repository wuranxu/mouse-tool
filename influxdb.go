package mouse_tool

import (
	"context"
	"errors"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/query"
	"time"
)

var (
	ErrorPingInfluxdb = errors.New("failed to connect influxdb")
)

type InfluxdbClient struct {
	token  string
	bucket string
	host   string
	org    string
	client influxdb2.Client
	writer api.WriteAPIBlocking
	reader api.QueryAPI
}

func NewInfluxdbClient(host, token, org, bucket string) (*InfluxdbClient, error) {
	client := influxdb2.NewClient(host, token)
	ping, err := client.Ping(context.TODO())
	if err != nil {
		return nil, err
	}
	if !ping {
		return nil, ErrorPingInfluxdb
	}
	db := &InfluxdbClient{
		token:  token,
		bucket: bucket,
		host:   host,
		org:    org,
		client: client,
	}
	db.writer = client.WriteAPIBlocking(org, bucket)
	db.reader = client.QueryAPI(org)
	return db, nil
}

func (i *InfluxdbClient) Write(measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	return i.WriteWithContext(context.Background(), measurement, tags, fields, ts)
}

// WriteWithContext write point to influxdb
func (i *InfluxdbClient) WriteWithContext(ctx context.Context, measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	p := influxdb2.NewPoint(measurement, tags, fields, ts)
	return i.writer.WritePoint(ctx, p)
}

// Query data
func (i *InfluxdbClient) Query(ctx context.Context, sql string) ([]*query.FluxRecord, error) {
	ans := make([]*query.FluxRecord, 0)
	result, err := i.reader.Query(ctx, sql)
	if err != nil {
		return ans, err
	}
	for result.Next() {
		ans = append(ans, result.Record())
	}
	return ans, nil
}

func (i *InfluxdbClient) Close() {
	i.client.Close()
}
