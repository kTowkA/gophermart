package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/storage"
	mocks "github.com/kTowkA/gophermart/internal/storage/mocs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var port = 8188

type Test struct {
	name           string
	body           any
	wantStatusCode int
}
type AppTestSuite struct {
	suite.Suite
	app         *AppServer
	mockStorage *mocks.Storage
}

func (suite *AppTestSuite) SetupSuite() {
	mockStorage := new(mocks.Storage)
	app, err := NewAppServer(config.Config{
		AddressApp: fmt.Sprintf(":%d", port),
	}, mockStorage)
	suite.Require().NoError(err)
	suite.app = app
	suite.mockStorage = mockStorage
	go suite.app.Start(context.TODO())
	time.Sleep(1 * time.Second)
}
func (suite *AppTestSuite) TearDownSuite() {
	suite.mockStorage.On("Close", mock.Anything).Return(nil)
	err := suite.app.storage.Close(context.Background())
	suite.Require().NoError(err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *AppTestSuite) TestCheckContentType() {
	type testCase struct {
		name             string
		contentType      string
		URL              string
		isBadContentType bool
	}
	tests := []testCase{
		{
			name:             "главный правильный",
			contentType:      "application/json",
			URL:              "/",
			isBadContentType: false,
		},
		{
			name:             "главный неправильный",
			contentType:      "application/xml",
			URL:              "/",
			isBadContentType: true,
		},
		{
			name:             "login правильный",
			contentType:      "application/json",
			URL:              "/api/user/login",
			isBadContentType: false,
		},
		{
			name:             "login неправильный",
			contentType:      "plain/text",
			URL:              "/api/user/login",
			isBadContentType: true,
		},
		{
			name:             "unknow неправильный",
			contentType:      "plain/text",
			URL:              "/api/user/unknow",
			isBadContentType: true,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := resty.New()
	for _, t := range tests {
		resp, err := client.R().
			SetContext(ctx).
			SetHeader("content-Type", t.contentType).
			Get("http://localhost" + suite.app.config.AddressApp + t.URL)
		suite.Assert().NoError(err)
		if t.isBadContentType {
			suite.Assert().EqualValues(http.StatusBadRequest, resp.StatusCode())
			continue
		}
		suite.Assert().NotEqualValues(http.StatusBadRequest, resp.StatusCode())
	}
}

func (suite *AppTestSuite) TestRegister() {

	suite.mockStorage.On("SaveUser", mock.Anything, "login-valid", mock.AnythingOfType("string")).Return(uuid.New(), nil)
	suite.mockStorage.On("SaveUser", mock.Anything, "login-is_used", mock.AnythingOfType("string")).Return(uuid.New(), storage.ErrLoginIsUsed)
	suite.mockStorage.On("SaveUser", mock.Anything, "login-error", mock.AnythingOfType("string")).Return(uuid.New(), errors.New("database error"))

	tests := []Test{
		{
			name:           "пустое тело запроса",
			body:           nil,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "невалидное тело запроса",
			body: `
			{
				login": "<login>",
				"password": "<password>"
			}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "успешно",
			body: `
			{
				"login": "login-valid",
				"password": "password"
			}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "внутренняя ошибка БД",
			body: `
			{
				"login": "login-error",
				"password": "password"
			}`,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "логин повторяется",
			body: `
			{
				"login": "login-is_used",
				"password": "password"
			}`,
			wantStatusCode: http.StatusConflict,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := resty.New().
		SetHeader("content-type", "application/json").
		SetBaseURL("http://localhost" + suite.app.config.AddressApp)
	for _, t := range tests {
		resp, err := client.R().
			SetContext(ctx).
			SetBody(t.body).
			Post("/api/user/register")
		suite.NoError(err)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode())
	}
	suite.mockStorage.AssertExpectations(suite.T())
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
