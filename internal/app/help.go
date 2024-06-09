package app

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// validLuhnNumber проверка числа в строке ccnS по алгоритму Луна
// в случае прохождения проверки возвращает конвертированное из строки число и true, в ином случае 0 и false
func validLuhnNumber(ccnS string) (int64, bool) {
	ccnS = strings.TrimSpace(ccnS)
	if ccnS == "" {
		return 0, false
	}
	sum := 0
	length := utf8.RuneCountInString(ccnS)
	parity := length % 2
	for i, char := range ccnS {
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
	number, _ := strconv.ParseInt(ccnS, 10, 64)
	return number, true
}
