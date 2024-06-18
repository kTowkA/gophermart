package storage

import "errors"

var (
	ErrLoginIsUsed                 = errors.New("такой логин уже занят")
	ErrUserNotFound                = errors.New("такого пользователя не существует")
	ErrOrderWasUploadByAnotherUser = errors.New("заказ с таким номером был загружен другим пользователем")
	ErrOrderWasAlreadyUpload       = errors.New("заказ уже был загружен пользователем")
	ErrOrdersNotFound              = errors.New("заказов не найдено")
	ErrWithdrawalsNotFound         = errors.New("пользователь еще не производил списания")
	ErrWithdrawNotEnough           = errors.New("пользователю не хватает средств для списания")
	ErrNothingHasBeenDone          = errors.New("данные уже актуальны")
)

// ErrorWithHttpStatus содержит ошибку базы данных и рекомендуемый ей http status код
type ErrorWithHTTPStatus struct {
	StorageError error
	HTTPStatus   int
}
