package app

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/stretchr/testify/suite"
)

var port = 8188

type AppTestSuite struct {
	suite.Suite
	app *AppServer
}

func (suite *AppTestSuite) SetupSuite() {
	app, err := NewAppServer(config.Config{
		AddressApp: fmt.Sprintf(":%d", port),
	})
	suite.Require().NoError(err)
	suite.app = app
	go suite.app.Start(context.TODO())
	time.Sleep(1 * time.Second)
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

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(AppTestSuite))
}
