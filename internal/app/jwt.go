// в этом файле содержаться функции относящиеся к созданию jwt токена, его проверки и парсингу
package app

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// cookieTokenName название ключа с токеном в куках
var cookieTokenName = "app_token"

// Claims — структура утверждений, которая включает стандартные утверждения и одно пользовательское UserClaims
type Claims struct {
	jwt.RegisteredClaims
	UserClaims
}

// UserClaims дополнительные утверждения. Логин нигде не используется, но пускай будет с заделом на будущее
type UserClaims struct {
	UserID uuid.UUID
	Login  string
}

// buildJWTString создаёт токен и возвращает его в виде строки.
func buildJWTString(userID uuid.UUID, userLogin, appSecret string, dur time.Duration) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
		},
		// собственное утверждение
		UserClaims: UserClaims{
			UserID: userID,
			Login:  userLogin,
		},
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(appSecret))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// getUserClaimsFromToken - получает UserClaims из JWT токена
func getUserClaimsFromToken(tokenString, secret string) (UserClaims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
	if err != nil {
		return UserClaims{}, err
	}

	if !token.Valid {
		return UserClaims{}, fmt.Errorf("токен не прошел проверку")
	}

	return claims.UserClaims, nil
}
