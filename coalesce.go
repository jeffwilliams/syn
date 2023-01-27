package syn

import (
	"bytes"
	"fmt"
)

type coalescer struct {
	it       Iterator
	accum    Token
	accumSet bool
}

func coalesce(in Iterator) Iterator {
	return &coalescer{
		it: in,
	}
}

func (c *coalescer) Next() (tok Token, err error) {
	if c.accumSet && (c.accum.Type == EOFType || c.accum.Type == Error) {
		return c.accum, nil
	}

	for {
		tok, err = c.it.Next()
		if err != nil || c.accum.Type == EOFType {
			return
		}

		if !c.accumSet {
			c.accum = tok
			c.accumSet = true
			continue
		}

		if c.accum.Type == tok.Type {
			c.merge(&tok)
			continue
		}

		// Type has changed. Return what we've accumulated and start
		// accumulating on top of the new token
		c.accum, tok = tok, c.accum
		return
	}
}

func (c *coalescer) merge(tok *Token) {
	c.accum.End = tok.End
	c.accum.Value = c.accum.Value[0:c.accum.Length()]
}

func (c *coalescer) State() IteratorState {
	return &coalescerState{
		accum:     c.accum,
		accumSet:  c.accumSet,
		iterState: c.it.State(),
	}
}

func (c *coalescer) SetState(s IteratorState) {
	state := s.(*coalescerState)

	c.accum = state.accum
	c.accumSet = state.accumSet
	c.it.SetState(state.iterState)
}

func (c *coalescer) SetStateWhenTextChanged(s IteratorState) {
	// TODO: Here we would move backwards to reparse the token
	// we've buffered so far. However we also need to move
	// back all the sub iterators as well.
	// This is temporarily done in Lexer.TokeniseAt for now.
	/*
		if c.accumSet {
			c.accumSet = false
			c.AddToIndex(-c.accum.Length())
		}
	*/
}

type coalescerState struct {
	accum     Token
	accumSet  bool
	iterState IteratorState
}

func (c coalescerState) Equal(o IteratorState) bool {
	// We only care if the underlying lexerStates are equal; that should be sufficient
	// to tell if the lexer can continue from this point.
	other, ok := o.(*coalescerState)
	if !ok {
		return false
	}

	return c.iterState.Equal(other.iterState)
}

func (c *coalescerState) SetIndex(i int) {
	c.iterState.SetIndex(i)
}

func (c *coalescerState) AddToIndex(i int) {
	c.iterState.AddToIndex(i)
}

func (c coalescerState) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "coalescerState: \n")
	fmt.Fprintf(&buf, "  accum: %s\n", c.accum)
	fmt.Fprintf(&buf, "  accumSet: %v\n", c.accumSet)

	s, ok := c.iterState.(fmt.Stringer)
	if ok {
		fmt.Fprintf(&buf, "  iterState:\n%s", s.String())
	}

	return buf.String()
}
