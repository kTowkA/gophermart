package storage

import "github.com/kTowkA/gophermart/internal/model"

var (
	StatusUndefined  = model.NewStatus(0, "UNDEFINED")
	StatusNew        = model.NewStatus(0, "NEW")
	StatusRegistered = model.NewStatus(0, "REGISTERED")
	StatusInvalid    = model.NewStatus(0, "INVALID")
	StatusProcessing = model.NewStatus(0, "PROCESSING")
	StatusProcessed  = model.NewStatus(0, "PROCESSED")
)

func StatusByValue(val string) model.Status {
	switch val {
	case "NEW":
		return StatusNew
	case "REGISTERED":
		return StatusRegistered
	case "INVALID":
		return StatusInvalid
	case "PROCESSING":
		return StatusProcessing
	case "PROCESSED":
		return StatusProcessed
	default:
		return StatusUndefined
	}
}
