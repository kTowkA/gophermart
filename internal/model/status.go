package model

type Status string

const (
	StatusNew        = Status("NEW")
	StatusProcessing = Status("PROCESSING")
	StatusInvalid    = Status("INVALID")
	StatusProcessed  = Status("PROCESSED")
)
