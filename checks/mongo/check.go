package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	defaultTimeoutConnect    = 5 * time.Second
	defaultTimeoutDisconnect = 5 * time.Second
	defaultTimeoutPing       = 5 * time.Second
)

// Config is the MongoDB checker configuration settings container.
type Config struct {
	// DSN is the MongoDB instance connection DSN. Optional if Client is supplied.
	DSN string

	// Client is an existing mongo client and can be used in place of DSN. Recommended,
	// since it reuses the application's connection pool instead of establishing and
	// tearing down a new connection on every check. The caller is responsible for
	// managing the client's lifecycle. Optional if DSN is supplied.
	Client *mongo.Client

	// TimeoutConnect defines timeout for establishing mongo connection, if not set - default value is used
	TimeoutConnect time.Duration
	// TimeoutDisconnect defines timeout for closing connection, if not set - default value is used
	TimeoutDisconnect time.Duration
	// TimeoutDisconnect defines timeout for making ping request, if not set - default value is used
	TimeoutPing time.Duration
}

// New creates new MongoDB health check that verifies the following:
// - connection establishing (skipped when an existing Client is supplied)
// - doing the ping command
func New(config Config) func(ctx context.Context) error {
	if config.TimeoutConnect == 0 {
		config.TimeoutConnect = defaultTimeoutConnect
	}

	if config.TimeoutDisconnect == 0 {
		config.TimeoutDisconnect = defaultTimeoutDisconnect
	}

	if config.TimeoutPing == 0 {
		config.TimeoutPing = defaultTimeoutPing
	}

	return func(ctx context.Context) (checkErr error) {
		shutdown, client, err := initClient(ctx, config)
		if err != nil {
			checkErr = err
			return
		}

		defer func() {
			// override checkErr only if there were no other errors
			if err := shutdown(ctx); err != nil && checkErr == nil {
				checkErr = err
			}
		}()

		ctxPing, cancelPing := context.WithTimeout(ctx, config.TimeoutPing)
		defer cancelPing()

		err = client.Ping(ctxPing, readpref.Primary())
		if err != nil {
			checkErr = fmt.Errorf("mongoDB health check failed on ping: %w", err)
			return
		}

		return
	}
}

func initClient(ctx context.Context, c Config) (func(ctx context.Context) error, *mongo.Client, error) {
	if c.Client != nil {
		return func(context.Context) error { return nil }, c.Client, nil
	}

	if c.DSN == "" {
		return nil, nil, errors.New("mongoDB DSN or an existing client is required to initialize mongoDB health check")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(c.DSN))
	if err != nil {
		return nil, nil, fmt.Errorf("mongoDB health check failed on client creation: %w", err)
	}

	ctxConn, cancelConn := context.WithTimeout(ctx, c.TimeoutConnect)
	defer cancelConn()

	err = client.Connect(ctxConn)
	if err != nil {
		return nil, nil, fmt.Errorf("mongoDB health check failed on connect: %w", err)
	}

	shutdown := func(ctx context.Context) error {
		ctxDisc, cancelDisc := context.WithTimeout(ctx, c.TimeoutDisconnect)
		defer cancelDisc()

		if err := client.Disconnect(ctxDisc); err != nil {
			return fmt.Errorf("mongoDB health check failed on closing connection: %w", err)
		}

		return nil
	}

	return shutdown, client, nil
}
