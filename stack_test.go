package syn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStack(t *testing.T) {
	assert := assert.New(t)

	s := NewStack()
	assert.Equal(0, s.Len())
	s.Pop(1)
	assert.Nil(s.Top())
	assert.Equal(0, s.Len())

	r := RuleSequence{}
	s.Push(r)
	assert.Equal(1, s.Len())
	assert.Equal(r, s.Top())
	s.Pop(1)
	assert.Equal(0, s.Len())

	r2 := RuleSequence{}
	s.Push(r)
	s.Push(r2)
	assert.Equal(2, s.Len())
	assert.Equal(r2, s.Top())
	s.Pop(2)
	assert.Equal(0, s.Len())
	assert.Nil(s.Top())
}
