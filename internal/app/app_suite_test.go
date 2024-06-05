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
	"github.com/kTowkA/gophermart/internal/model"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, _, err := suite.LoggedClient(ctx, "login orders post", "password", "TestRouteOrdersPost")
	suite.Require().NoError(err)

	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(49927398716)).Return(nil)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(5062821234567892)).Return(storage.ErrOrderWasAlreadyUpload)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, int64(1234561239)).Return(storage.ErrOrderWasUploadByAnotherUser)
	tests := []Test{
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

	for _, t := range tests {
		resp, err := client.R().SetContext(ctx).SetBody(t.body).SetHeader("content-type", t.contentType).Post(t.path)
		suite.NoError(err, t.name)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode(), t.name)
	}
}

// LoggedClient получаем авторизованного клиента
func (suite *AppTestSuite) LoggedClient(ctx context.Context, login, password string, called string) (*resty.Client, *resty.Response, error) {

	hashTestPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	suite.NoError(err)

	suite.mockStorage.On("PasswordHash", mock.Anything, login).Return(uuid.New(), string(hashTestPassword), nil)

	client := resty.
		New().
		SetBaseURL("http://localhost" + suite.app.config.AddressApp)
	resp, err := client.
		R().SetContext(ctx).
		SetBody(`{"login":"`+login+`","password":"`+password+`"}`).
		SetHeader("content-type", "application/json").
		Post("/api/user/login")
	suite.NoError(err, "logged client", "called: "+called)
	suite.EqualValues(http.StatusOK, resp.StatusCode(), "logged client", "called: "+called)
	return client, resp, err
}

// RouteOrdersGetV1 первый вариант - у пользователя нет заказов
func (suite *AppTestSuite) RouteOrdersGetV1(ctx context.Context) {

	client, resp, err := suite.LoggedClient(ctx, "login orders get 1", "password", "RouteOrdersGetV1")
	suite.Require().NoError(err)

	var userClaims UserClaims
	for _, c := range resp.Cookies() {
		if c.Name == cookieTokenName {
			userClaims, err = getUserClaimsFromToken(c.Value, suite.app.config.Secret)
			suite.NoError(err)
			break
		}
	}
	suite.NotEqualValues(UserClaims{}, userClaims, "userClaims")
	suite.mockStorage.On("Orders", mock.Anything, userClaims.UserID).Return(nil, storage.ErrOrdersNotFound)

	resp, err = client.
		R().SetContext(ctx).
		Get("/api/user/orders")
	suite.NoError(err, "orders get not orders")
	suite.EqualValues(http.StatusNoContent, resp.StatusCode(), "orders get not orders")
}

// RouteOrdersGetV2 второй вариант - произошла ошибка при запросе к БД
func (suite *AppTestSuite) RouteOrdersGetV2(ctx context.Context) {
	client, resp, err := suite.LoggedClient(ctx, "login orders get 2", "password", "RouteOrdersGetV2")
	suite.Require().NoError(err)

	userClaims := UserClaims{}
	for _, c := range resp.Cookies() {
		if c.Name == cookieTokenName {
			userClaims, err = getUserClaimsFromToken(c.Value, suite.app.config.Secret)
			suite.NoError(err)
			break
		}
	}
	suite.NotEqualValues(UserClaims{}, userClaims, "userClaims")
	suite.mockStorage.On("Orders", mock.Anything, userClaims.UserID).Return(nil, errors.New("get orders error"))

	resp, err = client.
		R().SetContext(ctx).
		Get("/api/user/orders")
	suite.NoError(err, "orders get error")
	suite.EqualValues(http.StatusInternalServerError, resp.StatusCode(), "orders get error")
}

// RouteOrdersGetV3 третий вариант - есть данные
func (suite *AppTestSuite) RouteOrdersGetV3(ctx context.Context) {
	client, resp, err := suite.LoggedClient(ctx, "login orders get 3", "password", "RouteOrdersGetV3")
	suite.Require().NoError(err)

	userClaims := UserClaims{}
	for _, c := range resp.Cookies() {
		if c.Name == cookieTokenName {
			userClaims, err = getUserClaimsFromToken(c.Value, suite.app.config.Secret)
			suite.NoError(err)
			break
		}
	}
	suite.NotEqualValues(UserClaims{}, userClaims, "userClaims")
	time1 := time.Now().Add(-1 * time.Hour)
	time2 := time.Now()
	// пропадают наносекунды при конвертировании, делаем так
	timeRFC3339 := time1.Format(time.RFC3339)
	time1, _ = time.Parse(time.RFC3339, timeRFC3339)
	timeRFC3339 = time2.Format(time.RFC3339)
	time2, _ = time.Parse(time.RFC3339, timeRFC3339)

	vals := model.ResponseOrders{
		{
			OrderNumber: 1,
			Status:      model.StatusNew,
			UploadedAt:  time1,
		},
		{
			OrderNumber: 2,
			Status:      model.StatusProcessed,
			Accrual:     100,
			UploadedAt:  time2,
		},
	}
	suite.
		mockStorage.
		On("Orders", mock.Anything, userClaims.UserID).
		Return(
			vals,
			nil,
		)
	result := model.ResponseOrders{}
	resp, err = client.
		R().SetContext(ctx).
		SetResult(&result).
		Get("/api/user/orders")
	suite.NoError(err, "orders get values")
	suite.EqualValues(http.StatusOK, resp.StatusCode(), "orders get values")
	suite.EqualValues(vals, result, "orders get values")
}

// TestRouteOrdersGet проверяем заказы пользователя
// так как при запросе используется ID пользователя, то для разных вариантов написаны функции RouteOrdersGetV(1.2.3)
func (suite *AppTestSuite) TestRouteOrdersGet() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	suite.RouteOrdersGetV1(ctx)
	suite.RouteOrdersGetV2(ctx)
	suite.RouteOrdersGetV3(ctx)
}
func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
