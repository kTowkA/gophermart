package model

import (
	"encoding/json"
	"strconv"
	"time"
)

func (o ResponseOrder) MarshalJSON() ([]byte, error) {
	// чтобы избежать рекурсии при json.Marshal, объявляем новый тип
	type ResponseOrderAlias ResponseOrder

	aliasValue := struct {
		ResponseOrderAlias
		// переопределяем поля внутри анонимной структуры
		OrderNumber string `json:"number"`
		UploadedAt  string `json:"uploaded_at"`
	}{
		// встраиваем значение всех полей изначального объекта (embedding)
		ResponseOrderAlias: ResponseOrderAlias(o),
		// задаём значение для переопределённого поля
		OrderNumber: strconv.FormatInt(int64(o.OrderNumber), 10),
		UploadedAt:  o.UploadedAt.Format(time.RFC3339),
	}

	return json.Marshal(aliasValue) // вызываем стандартный Marshal
}
func (o *ResponseOrder) UnmarshalJSON(data []byte) (err error) {
	// чтобы избежать рекурсии при json.Unmarshal, объявляем новый тип
	type ResponseOrderAlias ResponseOrder

	aliasValue := &struct {
		*ResponseOrderAlias
		// переопределяем поле внутри анонимной структуры
		OrderNumber string `json:"number"`
		UploadedAt  string `json:"uploaded_at"`
	}{
		ResponseOrderAlias: (*ResponseOrderAlias)(o),
	}
	// вызываем стандартный Unmarshal
	if err = json.Unmarshal(data, aliasValue); err != nil {
		return err
	}
	on, err := strconv.ParseInt(aliasValue.OrderNumber, 10, 64)
	if err != nil {
		return err
	}
	o.OrderNumber = OrderNumber(on)
	o.UploadedAt, err = time.Parse(time.RFC3339, aliasValue.UploadedAt)
	if err != nil {
		return err
	}
	return
}
