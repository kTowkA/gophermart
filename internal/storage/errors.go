package storage

import "errors"

var (
	ErrLoginIsUsed                 = errors.New("такой логин уже занят")
	ErrUserNotFound                = errors.New("такого пользователя не существует")
	ErrOrderWasUploadByAnotherUser = errors.New("заказ с таким номером был загружен другим пользователем")
	ErrOrderWasAlreadyUpload       = errors.New("заказ уже был загружен пользователем")
	ErrOrdersNotFound              = errors.New("пользователь еще не создал ни одного заказа")
	ErrWithdrawalsNotFound         = errors.New("пользователь еще не производил списания")
)
