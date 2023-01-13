package syn

type coalescer struct {
	it       Iterator
	accum    Token
	accumSet bool
}

func Coalesce(in Iterator) Iterator {
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
		if err != nil {
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

func (c *coalescer) State() interface{} {
	return coalescerState{
		accum:     c.accum,
		accumSet:  c.accumSet,
		iterState: c.it.State(),
	}
}

func (c *coalescer) SetState(s interface{}) {
	state := s.(coalescerState)

	c.accum = state.accum
	c.accumSet = state.accumSet
	c.it.SetState(state.iterState)
}

type coalescerState struct {
	accum     Token
	accumSet  bool
	iterState interface{}
}
