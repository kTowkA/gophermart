package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/kTowkA/gophermart/internal/storage"
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

	var connString string
	err = pool.Retry(func() error {
		time.Sleep(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		connString = fmt.Sprintf("postgres://user:pass@localhost:%s/user?sslmode=disable", resource.GetPort("5432/tcp"))
		conn, err := pgx.Connect(ctx, connString)
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
	ps, err := New(context.Background(), connString, mlog)
	suite.Require().NoError(err)
	suite.pstorage = ps

}
func (suite *PStorageTestSuite) TearDownSuite() {
	err := suite.pstorage.Close()
	suite.NoError(err)
	err = suite.clear.pool.Purge(suite.clear.resource)
	suite.NoError(err)
}
func (suite *PStorageTestSuite) TestSaveUser() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login := "login-test-save-user"
	userID, err := suite.pstorage.SaveUser(ctx, login, "pwd")
	suite.NoError(err)
	suite.NotEqualValues(uuid.UUID{}, userID)
	userIDtmp, err := suite.pstorage.SaveUser(ctx, login, "pwd")
	suite.ErrorIs(err, storage.ErrLoginIsUsed)
	suite.EqualValues(uuid.UUID{}, userIDtmp)
}
func (suite *PStorageTestSuite) TestUserID() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login, _, userID := suite.generateUser()
	actuserID, err := suite.pstorage.UserID(ctx, login)
	suite.NoError(err)
	suite.EqualValues(userID, actuserID)
	actuserID, err = suite.pstorage.UserID(ctx, login+"_test_userID")
	suite.ErrorIs(err, storage.ErrUserNotFound)
	suite.EqualValues(uuid.UUID{}, actuserID)
}
func (suite *PStorageTestSuite) TestHashPassword() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, hash, userID := suite.generateUser()
	actHash, err := suite.pstorage.HashPassword(ctx, userID)
	suite.NoError(err)
	suite.EqualValues(hash, actHash)
	actHash, err = suite.pstorage.HashPassword(ctx, uuid.New())
	suite.ErrorIs(err, storage.ErrUserNotFound)
	suite.Empty(actHash)
}

// generateUser создает нового пользователя. возвращает логин,пароль, userID
func (suite *PStorageTestSuite) generateUser() (string, string, uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login := uuid.New().String()
	hash := uuid.New().String()
	userID, err := suite.pstorage.SaveUser(ctx, login, hash)
	suite.Require().NoError(err)
	return login, hash, userID
}
func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(PStorageTestSuite))
}
