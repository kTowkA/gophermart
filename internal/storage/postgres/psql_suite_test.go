package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
)

type PStorageTestSuite struct {
	suite.Suite
	pstorage *PStorage
	clear    dockerClear
}
type dockerClear struct {
	resource *dockertest.Resource
	pool     *dockertest.Pool
}

func (suite *PStorageTestSuite) SetupSuite() {

	pool, err := dockertest.NewPool("")
	suite.Require().NoError(err)

	err = pool.Client.Ping()
	suite.Require().NoError(err)

	resource, err := pool.Run("postgres", "16", []string{"POSTGRES_USER=user", "POSTGRES_PASSWORD=pass"})
	suite.Require().NoError(err)

	err = pool.Retry(func() error {
		time.Sleep(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, fmt.Sprintf("postgres://user:pass@localhost:%s/user?sslmode=disable", resource.GetPort("5432/tcp")))
		suite.Require().NoError(err)
		defer conn.Close(ctx)
		return conn.Ping(ctx)
	})
	suite.Require().NoError(err)

	suite.clear = dockerClear{
		resource: resource,
		pool:     pool,
	}
	// ---------------------------------------------------------------------------------------------------
	mlog, err := logger.New()
	suite.Require().NoError(err)
	ps, err := New(context.Background(), "postgres://user:pass@localhost:5432/user?sslmode=disable", mlog)
	suite.Require().NoError(err)
	suite.pstorage = ps

}
func (suite *PStorageTestSuite) TearDownSuite() {
	err := suite.pstorage.Close()
	suite.NoError(err)
	err = suite.clear.pool.Purge(suite.clear.resource)
	suite.NoError(err)
}
func (suite *PStorageTestSuite) TestOne() {
	suite.NoError(nil)
}
func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(PStorageTestSuite))
}
