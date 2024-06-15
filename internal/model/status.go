package model

type Status struct {
	key   int
	value string
}

func NewStatus(key int, value string) Status {
	return Status{
		key:   key,
		value: value,
	}
}
func (s Status) Key() int {
	return s.key
}
func (s Status) Value() string {
	return s.value
}
func (s *Status) SetKey(key int) {
	s.key = key
}
func (s *Status) SetValue(value string) {
	s.value = value
}

// const (
// 	StatusNew        = Status("NEW")
// 	StatusProcessing = Status("PROCESSING")
// 	StatusInvalid    = Status("INVALID")
// 	StatusProcessed  = Status("PROCESSED")
// )
