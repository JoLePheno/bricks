package objstore

import (
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"github.com/minio/minio-go/v6"
	"github.com/pace/bricks/http/transport"
	"github.com/pace/bricks/maintenance/health/servicehealthcheck"
	"github.com/pace/bricks/maintenance/log"
	"github.com/prometheus/client_golang/prometheus"
)

type config struct {
	Endpoint        string `env:"S3_ENDPOINT" envDefault:"https://s3.amazonaws.com"`
	AccessKeyID     string `env:"S3_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY"`
	UseSSL          bool   `env:"S3_USE_SSL"`

	HealthCheckBucketName string        `env:"S3_HEALTH_CHECK_BUCKET_NAME" envDefault:"health-check"`
	HealthCheckObjectName string        `env:"S3_HEALTH_CHECK_OBJECT_NAME" envDefault:"latest.log"`
	HealthCheckResultTTL  time.Duration `env:"S3_HEALTH_CHECK_RESULT_TTL" envDefault:"2m"`
}

var cfg config

func init() {
	prometheus.MustRegister(paceObjStoreTotal)
	prometheus.MustRegister(paceObjStoreFailed)
	prometheus.MustRegister(paceObjStoreDurationSeconds)

	// parse log config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Failed to parse object storage environment: %v", err)
	}

	client, err := Client()
	if err != nil {
		log.Fatalf("Failed to create object storage client: %v", err)
	}
	servicehealthcheck.RegisterHealthCheck(&HealthCheck{
		Client: client,
	}, "objstore")
}

// Client with environment based configuration
func Client() (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.UseSSL)
	if err != nil {
		return nil, err
	}
	client.SetCustomTransport(newCustomTransport(cfg.Endpoint))
	return client, nil
}

// CustomClient with customized client
func CustomClient(endpoint string, opts *minio.Options) (*minio.Client, error) {
	client, err := minio.NewWithOptions(endpoint, opts)
	if err != nil {
		return nil, err
	}
	client.SetCustomTransport(newCustomTransport(endpoint))
	return client, nil
}

func newCustomTransport(endpoint string) http.RoundTripper {
	return transport.NewDefaultTransportChain().Use(newMetricRoundTripper(endpoint))
}

func newMetricRoundTripper(endpoint string) *metricRoundTripper {
	return &metricRoundTripper{
		endpoint: endpoint,
	}
}
