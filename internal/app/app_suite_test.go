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
	"github.com/kTowkA/gophermart/internal/logger"
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
	cancel      context.CancelFunc
}

func (suite *AppTestSuite) SetupSuite() {
	mlog, err := logger.New()
	suite.Require().NoError(err)
	mockStorage := new(mocks.Storage)
	app := NewAppServer(config.Config{
		AddressApp: fmt.Sprintf(":%d", port),
	}, mockStorage, mlog)
	suite.Require().NoError(err)
	suite.app = app
	suite.mockStorage = mockStorage
	suite.mockStorage.On("OrdersByStatuses", mock.Anything, []string{StatusUndefined, StatusNew, StatusProcessing}, 100, 0).Return(nil, storage.ErrOrdersNotFound)
	ctx, cancel := context.WithCancel(context.Background())
	suite.cancel = cancel
	go func() {
		err = suite.app.Start(ctx)
		suite.NoError(err)
	}()
	time.Sleep(1 * time.Second)
}
func (suite *AppTestSuite) TearDownSuite() {
	suite.cancel()
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
	userIDvalid, userIDnotFound := uuid.New(), uuid.New()
	validLogin, validLoginButUserIDnotFound, loginNotFound := "login-valid", "login-valid-2", "login-not-found"
	suite.mockStorage.On("UserID", mock.Anything, loginNotFound).Return(uuid.UUID{}, storage.ErrUserNotFound)
	suite.mockStorage.On("UserID", mock.Anything, validLoginButUserIDnotFound).Return(userIDnotFound, nil)
	suite.mockStorage.On("HashPassword", mock.Anything, userIDnotFound).Return("", storage.ErrUserNotFound)
	suite.mockStorage.On("UserID", mock.Anything, validLogin).Return(userIDvalid, nil)
	suite.mockStorage.On("HashPassword", mock.Anything, userIDvalid).Return(string(hashTestPassword), nil)
	tests := []Test{
		{
			name:           "пользовать не найден (по логину)",
			path:           "/api/user/login",
			body:           `{"login":"` + loginNotFound + `","password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "пользовать не найден (по id)",
			path:           "/api/user/login",
			body:           `{"login":"` + validLoginButUserIDnotFound + `","password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "пользовать успешно аутентифицирован",
			path:           "/api/user/login",
			body:           `{"login":"` + validLogin + `","password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "невалидный запрос (ошибка в JSON)",
			path:           "/api/user/login",
			body:           `{"login":` + validLogin + `,"password":"` + testPassword + `"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "пароль не совпадает",
			path:           "/api/user/login",
			body:           `{"login":"` + validLogin + `","password":"` + testPassword + `+1"}`,
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

	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, model.OrderNumber("49927398716")).Return(nil)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, model.OrderNumber("5062821234567892")).Return(storage.ErrOrderWasAlreadyUpload)
	suite.mockStorage.On("SaveOrder", mock.Anything, mock.Anything, model.OrderNumber("1234561239")).Return(storage.ErrOrderWasUploadByAnotherUser)
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
func (suite *AppTestSuite) LoggedClient(ctx context.Context, login, password string, called string) (*resty.Client, uuid.UUID, error) {

	hashTestPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	suite.NoError(err)

	userID := uuid.New()
	suite.mockStorage.On("UserID", mock.Anything, login).Return(userID, nil)
	suite.mockStorage.On("HashPassword", mock.Anything, userID).Return(string(hashTestPassword), nil)

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
	return client, userID, err
}

// RouteOrdersGetV1 первый вариант - у пользователя нет заказов
func (suite *AppTestSuite) RouteOrdersGetV1(ctx context.Context) {

	client, userID, err := suite.LoggedClient(ctx, "login orders get 1", "password", "RouteOrdersGetV1")
	suite.Require().NoError(err)

	suite.mockStorage.On("Orders", mock.Anything, userID).Return(nil, storage.ErrOrdersNotFound)

	resp, err := client.
		R().SetContext(ctx).
		Get("/api/user/orders")
	suite.NoError(err, "orders get not orders")
	suite.EqualValues(http.StatusNoContent, resp.StatusCode(), "orders get not orders")
}

// RouteOrdersGetV2 второй вариант - произошла ошибка при запросе к БД
func (suite *AppTestSuite) RouteOrdersGetV2(ctx context.Context) {
	client, userID, err := suite.LoggedClient(ctx, "login orders get 2", "password", "RouteOrdersGetV2")
	suite.Require().NoError(err)

	suite.mockStorage.On("Orders", mock.Anything, userID).Return(nil, errors.New("get orders error"))

	resp, err := client.
		R().SetContext(ctx).
		Get("/api/user/orders")
	suite.NoError(err, "orders get error")
	suite.EqualValues(http.StatusInternalServerError, resp.StatusCode(), "orders get error")
}

// RouteOrdersGetV3 третий вариант - есть данные
func (suite *AppTestSuite) RouteOrdersGetV3(ctx context.Context) {
	client, userID, err := suite.LoggedClient(ctx, "login orders get 3", "password", "RouteOrdersGetV3")
	suite.Require().NoError(err)

	time1 := time.Now().Add(-1 * time.Hour)
	time2 := time.Now()
	// пропадают наносекунды при конвертировании, делаем так
	timeRFC3339 := time1.Format(time.RFC3339)
	time1, _ = time.Parse(time.RFC3339, timeRFC3339)
	timeRFC3339 = time2.Format(time.RFC3339)
	time2, _ = time.Parse(time.RFC3339, timeRFC3339)

	vals := model.ResponseOrders{
		{
			OrderNumber: "1",
			Status:      model.StatusNew,
			UploadedAt:  time1,
		},
		{
			OrderNumber: "2",
			Status:      model.StatusProcessed,
			Accrual:     100,
			UploadedAt:  time2,
		},
	}
	suite.
		mockStorage.
		On("Orders", mock.Anything, userID).
		Return(
			vals,
			nil,
		)
	result := model.ResponseOrders{}
	resp, err := client.
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

func (suite *AppTestSuite) TestBalance() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// OK
	client, userID, err := suite.LoggedClient(ctx, "login-balance", "test", "TestBalance")
	suite.Require().NoError(err)
	balance := model.ResponseBalance{
		Current:   44,
		Withdrawn: 55,
	}
	suite.mockStorage.On("Balance", mock.Anything, userID).Return(balance, nil)
	result := model.ResponseBalance{}
	resp, err := client.R().SetContext(ctx).
		SetResult(&result).
		Get("/api/user/balance")
	suite.NoError(err)
	suite.EqualValues(http.StatusOK, resp.StatusCode())
	suite.EqualValues(balance, result)

	// Internal error
	client2, userID, err := suite.LoggedClient(ctx, "login-balance-2", "test", "TestBalance")
	suite.Require().NoError(err)
	balance = model.ResponseBalance{
		Current:   44,
		Withdrawn: 55,
	}
	suite.mockStorage.On("Balance", mock.Anything, userID).Return(balance, errors.New("balance error"))
	result = model.ResponseBalance{}
	resp, err = client2.R().SetContext(ctx).
		SetResult(&result).
		Get("/api/user/balance")
	suite.NoError(err)
	suite.EqualValues(http.StatusInternalServerError, resp.StatusCode())
	suite.EqualValues(model.ResponseBalance{}, result)
}
func (suite *AppTestSuite) TestWithdraw() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, userID, err := suite.LoggedClient(ctx, "login-withdraw", "test", "TestWithdraw")
	suite.Require().NoError(err)
	reqNotEnough := model.RequestWithdraw{
		OrderNumber: "49927398716",
		Sum:         1111.11,
	}
	reqErr := model.RequestWithdraw{
		OrderNumber: "49927398716",
		Sum:         666.666,
	}
	reqOK := model.RequestWithdraw{
		OrderNumber: "49927398716",
		Sum:         111.11,
	}
	suite.mockStorage.On("Withdraw", mock.Anything, userID, reqNotEnough).Return(storage.ErrWithdrawNotEnough)
	suite.mockStorage.On("Withdraw", mock.Anything, userID, reqErr).Return(errors.New("withdraw error"))
	suite.mockStorage.On("Withdraw", mock.Anything, userID, reqOK).Return(nil)
	tests := []Test{
		{
			name:           "неверный content-type",
			contentType:    "application/xml",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "ошибочное тело запроса",
			contentType:    "application/json",
			wantStatusCode: http.StatusBadRequest,
			body:           `{"order":123,"sum":123.123}`,
		},
		{
			name:           "неверный номер запроса",
			contentType:    "application/json",
			wantStatusCode: http.StatusUnprocessableEntity,
			body:           `{"order":"123","sum":123.123}`,
		},
		{
			name:           "недостаточно средств на балансе",
			contentType:    "application/json",
			wantStatusCode: http.StatusPaymentRequired,
			body:           reqNotEnough,
		},
		{
			name:           "ошщибка в сторадже",
			contentType:    "application/json",
			wantStatusCode: http.StatusInternalServerError,
			body:           reqErr,
		},
		{
			name:           "все хорошо",
			contentType:    "application/json",
			wantStatusCode: http.StatusOK,
			body:           reqOK,
		},
	}

	for _, t := range tests {
		resp, err := client.R().
			SetBody(t.body).
			SetHeader("content-type", t.contentType).
			Post("/api/user/balance/withdraw")
		suite.NoError(err, t.name)
		suite.EqualValues(t.wantStatusCode, resp.StatusCode())
	}
}
func (suite *AppTestSuite) TestWithdrawals() {
	ctxMain, cancelMain := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelMain()
	ctx200, cancel200 := context.WithCancel(ctxMain)
	defer cancel200()
	ctx204, cancel204 := context.WithCancel(ctxMain)
	defer cancel204()
	ctx500, cancel500 := context.WithCancel(ctxMain)
	defer cancel500()
	// OK 200
	client, userID, err := suite.LoggedClient(ctx200, "login-withdrawals-200", "test", "TestWithdrawals")
	suite.Require().NoError(err)
	time1 := time.Now().Add(-1 * time.Hour)
	time2 := time.Now().Add(-2 * time.Hour)
	// пропадают наносекунды при конвертировании, делаем так
	timeRFC3339 := time1.Format(time.RFC3339)
	time1, _ = time.Parse(time.RFC3339, timeRFC3339)
	timeRFC3339 = time2.Format(time.RFC3339)
	time2, _ = time.Parse(time.RFC3339, timeRFC3339)
	returnValue := model.ResponseWithdrawals{
		{
			OrderNumber: "111",
			Sum:         111.11,
			ProcessedAt: time1,
		},
		{
			OrderNumber: "222",
			Sum:         222.22,
			ProcessedAt: time2,
		},
	}
	suite.mockStorage.On("Withdrawals", mock.Anything, userID).Return(returnValue, nil)
	result := model.ResponseWithdrawals{}
	resp, err := client.R().SetContext(ctx200).
		SetResult(&result).
		Get("/api/user/withdrawals")
	suite.NoError(err)
	suite.EqualValues(http.StatusOK, resp.StatusCode())
	suite.EqualValues(returnValue, result)

	// no content 204
	client, userID, err = suite.LoggedClient(ctx204, "login-withdrawals-204", "test", "TestWithdrawals")
	suite.Require().NoError(err)
	suite.mockStorage.On("Withdrawals", mock.Anything, userID).Return(nil, storage.ErrWithdrawalsNotFound)
	result = model.ResponseWithdrawals{}
	resp, err = client.R().SetContext(ctx204).
		SetResult(&result).
		Get("/api/user/withdrawals")
	suite.NoError(err)
	suite.EqualValues(http.StatusNoContent, resp.StatusCode())
	suite.EqualValues(model.ResponseWithdrawals{}, result)
	// internal error 500
	client, userID, err = suite.LoggedClient(ctx500, "login-withdrawals-500", "test", "TestWithdrawals")
	suite.Require().NoError(err)
	suite.mockStorage.On("Withdrawals", mock.Anything, userID).Return(nil, errors.New("withdrawals error"))
	result = model.ResponseWithdrawals{}
	resp, err = client.R().SetContext(ctx500).
		SetResult(&result).
		Get("/api/user/withdrawals")
	suite.NoError(err)
	suite.EqualValues(http.StatusInternalServerError, resp.StatusCode())
	suite.EqualValues(model.ResponseWithdrawals{}, result)
}
func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
