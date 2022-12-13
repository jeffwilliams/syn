package syn

type Iterator interface {
	Next() (Token, error)
}

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
	if c.accumSet && c.accum.Typ == EOFType {
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

		if c.accum.Typ == tok.Typ {
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

func (c *coalescer) State() CoalescerState {
	return CoalescerState{
		accum:    c.accum,
		accumSet: c.accumSet,
	}
}

func (c *coalescer) SetState(s CoalescerState) {
	c.accum = s.accum
	c.accumSet = s.accumSet
}

type CoalescerState struct {
	accum    Token
	accumSet bool
}
