package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/storage"
	mocks "github.com/kTowkA/gophermart/internal/storage/mocs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

var port = 8188

type Test struct {
	name           string
	path           string
	method         string
	contentType    string
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

func (suite *AppTestSuite) TestMiddlewareCheckOnlyAuthUser() {
	// здесь не сохраняем куки между запросами
	suite.mockStorage.On("SaveUser", mock.Anything, "test-middleware-login", mock.AnythingOfType("string")).Return(uuid.New(), nil)
	tests := []Test{
		{
			name:           "разрешенный запрос всем пользователям",
			path:           "/api/user/register",
			method:         http.MethodPost,
			body:           `{"login":"test-middleware-login","password":"1"}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "запрос только для зарегистрированных пользователей",
			path:           "/api/user/balance",
			method:         http.MethodPost,
			body:           `{"login":"test-middleware-login","password":"1"}`,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "запрос только для зарегистрированных пользователей",
			path:           "/api/user/orders",
			method:         http.MethodGet,
			body:           `{"zzz":"zz","z":"z"}`,
			wantStatusCode: http.StatusUnauthorized,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, t := range tests {
		var (
			resp *resty.Response
			err  error
		)
		req := resty.New().
			SetHeader("content-type", "application/json").
			SetBaseURL("http://localhost" + suite.app.config.AddressApp).
			R().
			SetContext(ctx).
			SetBody(t.body)
		switch t.method {
		case http.MethodPost:
			resp, err = req.Post(t.path)
		case http.MethodGet:
			resp, err = req.Get(t.path)
		default:
			continue
		}
		suite.NoError(err, t.name)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode(), t.name)
	}
}
func (suite *AppTestSuite) TestRouteRegister() {

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
}
func (suite *AppTestSuite) TestRouteLogin() {
	testPassword := "test"
	hashTestPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	suite.NoError(err)
	suite.mockStorage.On("PasswordHash", mock.Anything, "test-login-valid").Return(uuid.New(), string(hashTestPassword), nil)
	tests := []Test{
		{
			name:           "пользовать успешно аутентифицирован",
			path:           "/api/user/login",
			body:           `{"login":"test-login-valid","password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "невалидный запрос",
			path:           "/api/user/login",
			body:           `{"login":test-login-valid,"password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "пароль не совпадает",
			path:           "/api/user/login",
			body:           `{"login":"test-login-valid","password":"` + testPassword + `+1"}`,
			wantStatusCode: http.StatusUnauthorized,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, t := range tests {
		resp, err := resty.New().
			SetHeader("content-type", "application/json").
			SetBaseURL("http://localhost" + suite.app.config.AddressApp).
			R().
			SetBody(t.body).
			SetContext(ctx).
			Post(t.path)
		suite.NoError(err, t.name)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode(), t.name)
	}
}

func (suite *AppTestSuite) TestRouteOrdersPost() {
	testPassword := "test"
	hashTestPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	suite.NoError(err)
	suite.mockStorage.On("PasswordHash", mock.Anything, "test-login-valid").Return(uuid.New(), string(hashTestPassword), nil)

	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(49927398716)).Return(nil)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(5062821234567892)).Return(storage.ErrOrderWasAlreadyUpload)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(1234561239)).Return(storage.ErrOrderWasUploadByAnotherUser)
	tests := []Test{
		{
			name:           "пользовать успешно аутентифицирован",
			path:           "/api/user/login",
			body:           `{"login":"test-login-valid","password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusOK,
			method:         http.MethodPost,
			contentType:    "application/json",
		},
		{
			name:           "ошибочный контент-тайп",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "application/json",
			wantStatusCode: http.StatusBadRequest,
			body:           "49927398716",
		},
		{
			name:           "пустое тело",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "text/plain",
			wantStatusCode: http.StatusUnprocessableEntity,
			body:           strings.NewReader(""),
		},
		{
			name:           "невалидный номер",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "text/plain",
			wantStatusCode: http.StatusUnprocessableEntity,
			body:           "499273987161",
		},
		{
			name:           "все хорошо",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "text/plain",
			wantStatusCode: http.StatusCreated,
			body:           "49927398716",
		},
		{
			name:           "уже был загружен",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "text/plain",
			wantStatusCode: http.StatusOK,
			body:           "5062821234567892",
		},
		{
			name:           "другой пользователь загрузил",
			path:           "/api/user/orders",
			method:         http.MethodPost,
			contentType:    "text/plain",
			wantStatusCode: http.StatusConflict,
			body:           "1234561239",
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := resty.
		New().
		SetBaseURL("http://localhost" + suite.app.config.AddressApp)
	for _, t := range tests {
		resp, err := client.R().SetContext(ctx).SetBody(t.body).SetHeader("content-type", t.contentType).Post(t.path)
		suite.NoError(err, t.name)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode(), t.name)
	}
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
