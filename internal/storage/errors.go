package storage

import "errors"

var (
	ErrLoginIsUsed                 = errors.New("такой логин уже занят")
	ErrUserNotFound                = errors.New("такого пользователя не существует")
	ErrOrderWasUploadByAnotherUser = errors.New("заказ с таким номером был загружен другим пользователем")
	ErrOrderWasAlreadyUpload       = errors.New("заказ уже был загружен пользователем")
)
