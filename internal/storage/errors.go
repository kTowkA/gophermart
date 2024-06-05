package storage

import "errors"

var (
	ErrLoginIsUsed  = errors.New("такой логин уже занят")
	ErrUserNotFound = errors.New("такого пользователя не существует")
)
