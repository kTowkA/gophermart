package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckLuhn(t *testing.T) {
	_, ok := validLuhnNumber("")
	assert.False(t, ok)
	_, ok = validLuhnNumber("123fasf")
	assert.False(t, ok)
	_, ok = validLuhnNumber("123")
	assert.False(t, ok)
	_, ok = validLuhnNumber("4561261212345464")
	assert.False(t, ok)
	_, ok = validLuhnNumber("4561261212345467")
	assert.True(t, ok)
	_, ok = validLuhnNumber("49927398716")
	assert.True(t, ok)
	_, ok = validLuhnNumber("499273987161")
	assert.False(t, ok)
}
