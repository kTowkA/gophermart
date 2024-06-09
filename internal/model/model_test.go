package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMarshalResponseOrder(t *testing.T) {
	t1 := time.Now()
	val := ResponseOrder{
		OrderNumber: "12345",
		Status:      StatusNew,
		Accrual:     11.1,
		UploadedAt:  t1,
	}
	expect := `{"number":"12345","status":"NEW","accrual":11.1,"uploaded_at":"` + t1.Format(time.RFC3339) + `"}`
	b, err := json.Marshal(val)
	assert.NoError(t, err)
	assert.JSONEq(t, expect, string(b))
	newVal := ResponseOrder{}
	err = json.Unmarshal([]byte(expect), &newVal)
	assert.NoError(t, err)

	// ну такое..приходится приводить к этому виду, так как пропадают наносекунды в формате RFC3339  и сравнение проваливается
	t1RFC3339 := val.UploadedAt.Format(time.RFC3339)
	val.UploadedAt, _ = time.Parse(time.RFC3339, t1RFC3339)

	assert.EqualValues(t, val, newVal, t1.Format(time.RFC3339))
}
