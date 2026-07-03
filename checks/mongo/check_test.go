package mongo

import (
	"context"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/stretchr/testify/require"
)

const mgDSNEnv = "HEALTH_GO_MG_DSN"

func TestNew(t *testing.T) {
	check := New(Config{
		DSN: getDSN(t),
	})

	err := check(context.Background())
	require.NoError(t, err)
}

func TestNew_withClient(t *testing.T) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(getDSN(t)))
	require.NoError(t, err)

	defer func() {
		errDisc := client.Disconnect(ctx)
		require.NoError(t, errDisc)
	}()

	check := New(Config{
		Client: client,
	})

	err = check(ctx)
	require.NoError(t, err)
}

func TestNewWithError(t *testing.T) {
	check := New(Config{})

	err := check(context.Background())
	require.Error(t, err)
}

func getDSN(t *testing.T) string {
	t.Helper()

	mongoDSN, ok := os.LookupEnv(mgDSNEnv)
	require.True(t, ok)

	// "docker compose port <service> <port>" returns 0.0.0.0:XXXX locally, change it to local port
	mongoDSN = strings.Replace(mongoDSN, "0.0.0.0:", "127.0.0.1:", 1)

	return mongoDSN
}
