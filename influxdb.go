package MetricsInflux

import (
	"context"
	"fmt"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"

	"github.com/yeencloud/lib-logger/domain"

	"github.com/yeencloud/lib-shared"
	"github.com/yeencloud/lib-shared/log"

	"github.com/yeencloud/lib-logger"
	"github.com/yeencloud/lib-metrics"
)

type InfluxConfig struct {
	Host string `config:"INFLUXDB_HOST" default:"localhost"`
	Port int    `config:"INFLUXDB_PORT" default:"8086"`

	Token        shared.Secret `config:"INFLUXDB_TOKEN"`
	Organization string        `config:"INFLUXDB_ORG"`
	Bucket       string        `config:"INFLUXDB_BUCKET"`
}

func (c *InfluxConfig) GetAddress() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

type Influx struct {
	config *InfluxConfig

	client influxdb2.Client

	serviceName shared.ServiceName

	writeAPI api.WriteAPIBlocking
}

func (i *Influx) Connect() error {
	client := influxdb2.NewClient(i.config.GetAddress(), i.config.Token.Value)
	i.client = client
	i.writeAPI = client.WriteAPIBlocking(i.config.Organization, i.config.Bucket)

	_, err := client.Ping(context.Background())
	if err != nil {
		return err
	}

	i.LogPoint(metrics.MetricPoint{
		Name: "service",
		Tags: map[string]string{
			"service": string(i.serviceName),
		},
	}, metrics.MetricValues{
		"start": 1,
	})

	return nil
}

func (i *Influx) newPoint(metricPoint metrics.MetricPoint) *write.Point {
	p := influxdb2.NewPointWithMeasurement(metricPoint.Name)

	for k, v := range metricPoint.Tags {
		p = p.AddTag(k, v)
	}
	p = p.AddTag("service", string(i.serviceName))

	return p
}

func (i *Influx) LogPoint(point metrics.MetricPoint, values metrics.MetricValues) {
	writer := i.writeAPI

	p := i.newPoint(point)

	for k, v := range values {
		p = p.AddField(k, v)
	}

	err := writer.WritePoint(context.Background(), p)

	if err != nil {
		Logger.Log(LoggerDomain.LogLevelWarn).WithField(log.Path{
			Identifier: "error",
		}, err).WithField(log.Path{
			Identifier: "point",
		}, point).WithField(log.Path{
			Identifier: "values",
		}, values).Msg("Failed to write point")
	}
}

func NewInflux(serviceName shared.ServiceName, config *InfluxConfig) (metrics.MetricsInterface, error) {
	return &Influx{
		config:      config,
		serviceName: serviceName,
	}, nil
}
