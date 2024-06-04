package storage

import "errors"

var (
	ErrLoginIsUsed = errors.New("такой логин уже занят")
)
