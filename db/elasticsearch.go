package db

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/elastic/go-elasticsearch/v8"
)

// BulkIndexer const
const NUM_WORKERS = 5
const FLUSH_BYTES = 5000000
const FLUSH_INTERVAL = time.Second * 30

// Client Connection
func NewConnectionEs() (*elasticsearch.Client, error) {
	retryBackoff := backoff.NewExponentialBackOff()
	protocol := "http"

	if settingsData.NODE_ENV == "prod" {
		protocol += "s"
	}

	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("%s://%s:%d", protocol, settingsData.ELS_HOST, settingsData.ELS_PORT),
		},
		Username: settingsData.ELS_USERNAME,
		Password: settingsData.ELS_PASSWORD,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			ResponseHeaderTimeout: time.Second * 2,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		// Retry on 429 TooManyRequests statuses
		RetryOnStatus: []int{502, 503, 504, 429},
		// Configure the backoff function
		RetryBackoff: func(attempt int) time.Duration {
			if attempt == 1 {
				retryBackoff.Reset()
			}
			return retryBackoff.NextBackOff()
		},
		MaxRetries: 5,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	fmt.Println(es.Info())
	return es, nil
}

// Construct Query
func ConstructQuery(q string) *strings.Reader {
	var query = `{"query": {`

	query += fmt.Sprintf("%s}}", q)

	var b strings.Builder
	b.WriteString(query)
	read := strings.NewReader(b.String())
	return read
}
