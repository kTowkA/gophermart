package luhn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckLuhn(t *testing.T) {
	_, ok := ValidateLuhnNumber("")
	assert.False(t, ok)
	_, ok = ValidateLuhnNumber("123fasf")
	assert.False(t, ok)
	_, ok = ValidateLuhnNumber("123")
	assert.False(t, ok)
	_, ok = ValidateLuhnNumber("4561261212345464")
	assert.False(t, ok)
	_, ok = ValidateLuhnNumber("4561261212345467")
	assert.True(t, ok)
	_, ok = ValidateLuhnNumber("49927398716")
	assert.True(t, ok)
	_, ok = ValidateLuhnNumber("499273987161")
	assert.False(t, ok)
}
