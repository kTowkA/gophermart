package model

import "time"

type RequestRegister struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RequestLogin struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type OrderNumber uint

type ResponseOrder struct {
	OrderNumber OrderNumber `json:"number"`
	Status      Status      `json:"status"`
	Accrual     float64     `json:"accrual,omitempty"`
	UploadedAt  time.Time   `json:"uploaded_at"`
}

type ResponseOrders []ResponseOrder

type ResponseBalance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type RequestWithdraw struct {
	OrderNumber OrderNumber `json:"order"`
	Sum         float64     `json:"sum"`
}

type ResponseWithdraw struct {
	OrderNumber OrderNumber `json:"order"`
	Sum         float64     `json:"sum"`
	ProcessedAt time.Time   `json:"processed_at"`
}

type ResponseWithdrawals []ResponseWithdraw

type ResponseBonusSystem struct {
	OrderNumber OrderNumber `json:"order"`
	Status      Status      `json:"status"`
	Accrual     float64     `json:"accrual,omitempty"`
}
