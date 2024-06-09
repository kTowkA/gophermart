package model

import (
	"encoding/json"
	"time"
)

func (o ResponseOrder) MarshalJSON() ([]byte, error) {
	// чтобы избежать рекурсии при json.Marshal, объявляем новый тип
	type ResponseOrderAlias ResponseOrder

	aliasValue := struct {
		ResponseOrderAlias
		// переопределяем поля внутри анонимной структуры
		UploadedAt string `json:"uploaded_at"`
	}{
		// встраиваем значение всех полей изначального объекта (embedding)
		ResponseOrderAlias: ResponseOrderAlias(o),
		// задаём значение для переопределённого поля
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
	}

	return json.Marshal(aliasValue) // вызываем стандартный Marshal
}
func (o *ResponseOrder) UnmarshalJSON(data []byte) (err error) {
	// чтобы избежать рекурсии при json.Unmarshal, объявляем новый тип
	type ResponseOrderAlias ResponseOrder

	aliasValue := &struct {
		*ResponseOrderAlias
		// переопределяем поле внутри анонимной структуры
		UploadedAt string `json:"uploaded_at"`
	}{
		ResponseOrderAlias: (*ResponseOrderAlias)(o),
	}
	// вызываем стандартный Unmarshal
	if err = json.Unmarshal(data, aliasValue); err != nil {
		return err
	}
	o.UploadedAt, err = time.Parse(time.RFC3339, aliasValue.UploadedAt)
	if err != nil {
		return err
	}
	return
}
func (w ResponseWithdraw) MarshalJSON() ([]byte, error) {
	// чтобы избежать рекурсии при json.Marshal, объявляем новый тип
	type ResponseWithdrawAlias ResponseWithdraw

	aliasValue := struct {
		ResponseWithdrawAlias
		// переопределяем поля внутри анонимной структуры
		ProcessedAt string `json:"processed_at"`
	}{
		// встраиваем значение всех полей изначального объекта (embedding)
		ResponseWithdrawAlias: ResponseWithdrawAlias(w),
		// задаём значение для переопределённого поля
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
	}

	return json.Marshal(aliasValue) // вызываем стандартный Marshal
}
func (w *ResponseWithdraw) UnmarshalJSON(data []byte) (err error) {
	// чтобы избежать рекурсии при json.Unmarshal, объявляем новый тип
	type ResponseWithdrawAlias ResponseWithdraw

	aliasValue := &struct {
		*ResponseWithdrawAlias
		// переопределяем поле внутри анонимной структуры
		ProcessedAt string `json:"processed_at"`
	}{
		ResponseWithdrawAlias: (*ResponseWithdrawAlias)(w),
	}
	// вызываем стандартный Unmarshal
	if err = json.Unmarshal(data, aliasValue); err != nil {
		return err
	}
	w.ProcessedAt, err = time.Parse(time.RFC3339, aliasValue.ProcessedAt)
	if err != nil {
		return err
	}
	return
}
