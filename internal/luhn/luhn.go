package luhn

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValidateLuhnNumber проверка числа в строке number по алгоритму Луна. В случае прохождения проверки возвращает конвертированное из строки число и true, в ином случае 0 и false. Взяли из википедии алгоритм на псевдокоде и переписали на go
func ValidateLuhnNumber(numberString string) (int64, bool) {
	numberString = strings.TrimSpace(numberString)
	if numberString == "" {
		return 0, false
	}
	sum := 0
	length := utf8.RuneCountInString(numberString)
	parity := length % 2
	for i, char := range numberString {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return 0, false
		}
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	if !(sum%10 == 0) {
		return 0, false
	}
	number, _ := strconv.ParseInt(numberString, 10, 64)
	return number, true
}
