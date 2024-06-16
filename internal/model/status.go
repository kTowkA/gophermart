package model

// статус. так как у нас в репозитории ключ и значение это не одно поле, делаем так
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
