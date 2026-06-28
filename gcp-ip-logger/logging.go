package main

import (
	"context"
	"log"

	"cloud.google.com/go/logging"
)

type logSink interface {
	log(ctx context.Context, entry IPLog)
	close() error
}

type cloudLogger struct {
	client *logging.Client
	sink   logSink
}

func newCloudLogger(ctx context.Context, projectID, logName string) (*cloudLogger, error) {
	if projectID == "" {
		log.Fatalf("GCP_PROJECT_ID is required (set env var or run on GCE/GKE with metadata server access)")
	}

	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	glogger := client.Logger(logName)
	return &cloudLogger{
		client: client,
		sink:   &gcpSink{logger: glogger},
	}, nil
}

func (c *cloudLogger) log(ctx context.Context, entry IPLog) {
	c.sink.log(ctx, entry)
}

func (c *cloudLogger) close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return c.sink.close()
}

type gcpSink struct {
	logger *logging.Logger
}

func (g *gcpSink) log(_ context.Context, entry IPLog) {
	g.logger.Log(logging.Entry{
		Payload: map[string]any{
			"ip":         entry.IP,
			"timestamp":  entry.Timestamp,
			"user_agent": entry.UserAgent,
			"path":       entry.Path,
		},
		Labels: map[string]string{
			"component": "ip-logger",
		},
	})
}

func (g *gcpSink) close() error { return nil }