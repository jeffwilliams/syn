package syn

import "fmt"

type offsetMap struct {
	transitions []int
}

func (o *offsetMap) push(i int) {
	o.transitions = append(o.transitions, i)
}

func (o *offsetMap) iterator() offsetIterator {
	return offsetIterator{transitions: o.transitions}
}

// offsetIterator is used to get the original length of tokens
// before sequence \r\n was converted to \n
type offsetIterator struct {
	transitions []int
	offset      int
}

// OriginalLen returns the original length of the token tok in runes.
// This function must be called with each token being iterated over in order.
// Only one of OriginalLen or OriginalLenRunes may be called on a single token
// in the steam.
func (o *offsetIterator) Offset() int {
	return o.offset
}

func (o *offsetIterator) Advance(length int) {
	newOffset := o.offset + length
	newOffset += o.transitionsCrossed(newOffset)
	o.offset = newOffset
}

func (o *offsetIterator) transitionsCrossed(newOffset int) int {
	c := 0
	for len(o.transitions) > 0 && o.transitions[0] < newOffset {
		c++
		newOffset++
		o.transitions = o.transitions[1:]
	}
	return c
}

func (o offsetIterator) Clone() offsetIterator {
	t := make([]int, len(o.transitions))
	copy(t, o.transitions)

	return offsetIterator{
		offset:      o.offset,
		transitions: t,
	}
}

// adjustForLF is an Iterator decorator that adjusts the values of the token's Start, End and Value
// to account for the modifications done by the function ensureLF; namely the conversion of
// \r\n to \n.
func adjustForLF(text []rune, it Iterator, offIt offsetIterator) Iterator {
	return &offsetAdjuster{
		text:       text,
		it:         it,
		offsetIter: offIt,
	}
}

type offsetAdjuster struct {
	text       []rune
	it         Iterator
	offsetIter offsetIterator
}

func (a *offsetAdjuster) Next() (tok Token, err error) {
	tok, err = a.it.Next()
	if err != nil {
		return
	}

	if tok.Type == EOFType || tok.Type == Error {
		return
	}

	l := tok.Length()
	tok.Start = a.offsetIter.Offset()
	a.offsetIter.Advance(l)
	if a.offsetIter.Offset() < tok.Start {
		err = fmt.Errorf("*offsetAdjuster.Next: offsetIter returned an offset that is invalid. "+
			"offset: %d tok: %s",
			a.offsetIter.Offset(), tok)
	}
	tok.End = a.offsetIter.Offset()
	tok.Value = a.text[tok.Start:tok.End]

	return
}

func (c *offsetAdjuster) State() interface{} {
	return offsetAdjusterState{
		iterState:  c.it.State(),
		offsetIter: c.offsetIter.Clone(),
		text:       c.text,
	}
}

func (c *offsetAdjuster) SetState(s interface{}) {
	state := s.(offsetAdjusterState)

	c.it.SetState(state.iterState)
	c.offsetIter = state.offsetIter.Clone()
	c.text = state.text
}

type offsetAdjusterState struct {
	iterState  interface{}
	offsetIter offsetIterator
	text       []rune
}
