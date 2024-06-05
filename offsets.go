package syn

import (
	"bytes"
	"fmt"
)

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
	transitions         []int
	nextTransitionIndex int
	offset              int
}

func (o *offsetIterator) Offset() int {
	return o.offset
}

func (o *offsetIterator) Advance(length int) {
	if length >= 0 {
		o.forward(length)
	} else {
		o.backward(length)
	}
}

func (o *offsetIterator) forward(length int) {
	newOffset := o.offset + length
	newOffset += o.transitionsCrossedForward(newOffset)
	o.offset = newOffset
}

func (o *offsetIterator) backward(length int) {
	newOffset := o.offset + length
	newOffset -= o.transitionsCrossedBackward(newOffset)
}

func (o *offsetIterator) transitionsCrossedForward(newOffset int) int {
	c := 0
	for o.nextTransitionIndex < len(o.transitions) && o.transitions[o.nextTransitionIndex] < newOffset {
		o.nextTransitionIndex++
		c++
		newOffset++
	}
	/*
		for len(o.transitions) > 0 && o.transitions[0] < newOffset {
			c++
			newOffset++
			o.transitions = o.transitions[1:]
		}*/
	return c
}

func (o *offsetIterator) transitionsCrossedBackward(newOffset int) int {
	c := 0
	next := o.nextTransitionIndex - 1
	for next >= 0 && o.transitions[next] > newOffset {
		next--
		c++
		newOffset--
	}
	/*
	   for len(o.transitions) > 0 && o.transitions[0] < newOffset {
	     c++
	     newOffset++
	     o.transitions = o.transitions[1:]
	   }*/
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

func (i offsetIterator) equal(o offsetIterator) bool {
	return i.transitionsEqual(o) &&
		i.offset == o.offset
}

func (iter offsetIterator) transitionsEqual(o offsetIterator) bool {
	for i, t := range iter.transitions {
		if t != o.transitions[i] {
			return false
		}
	}
	return true
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

	if tok.Type == EOFType {
		return
	}

	l := tok.Length()
	tok.Start = a.offsetIter.Offset()
	a.offsetIter.Advance(l)
	if a.offsetIter.Offset() < tok.Start {
		err = fmt.Errorf("*offsetAdjuster.Next: offsetIter returned an offset that is invalid: offset is before token start. "+
			"offset: %d tok: %s",
			a.offsetIter.Offset(), tok)
		return
	}
	if a.offsetIter.Offset() > len(a.text) {
		err = fmt.Errorf("*offsetAdjuster.Next: offsetIter returned an offset that is invalid: offset is >= text length. "+
			"offset: %d tok: '%s' tok length: %d text length: %d. Transitions: %v. Text: '%s'",
			a.offsetIter.Offset(), tok, tok.Length(), len(a.text), a.offsetIter.transitions, string(a.text))
		return
	}
	tok.End = a.offsetIter.Offset()
	tok.Value = a.text[tok.Start:tok.End]

	return
}

func (c *offsetAdjuster) State() IteratorState {
	return &offsetAdjusterState{
		iterState:  c.it.State(),
		offsetIter: c.offsetIter.Clone(),
		//text:       c.text,
	}
}

func (c *offsetAdjuster) SetState(s IteratorState) {
	state := s.(*offsetAdjusterState)

	c.it.SetState(state.iterState)
	c.offsetIter = state.offsetIter.Clone()
	//c.text = state.text // Don't set text because it might have changed
}

type offsetAdjusterState struct {
	iterState  IteratorState
	offsetIter offsetIterator
	text       []rune
}

func (s offsetAdjusterState) Equal(o IteratorState) bool {
	a, ok := o.(*offsetAdjusterState)
	if !ok {
		return false
	}

	return s.iterState.Equal(a.iterState) && s.offsetIter.equal(a.offsetIter)
}

func (s *offsetAdjusterState) SetIndex(i int) {
	s.offsetIter.offset = i
	s.iterState.SetIndex(i)
}

func (s *offsetAdjusterState) AddToIndex(i int) {
	s.offsetIter.offset += i
	s.iterState.AddToIndex(i)
	s.offsetIter.Advance(i)
}

func (s offsetAdjusterState) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "offsetAdjusterState: \n")
	fmt.Fprintf(&buf, "  text: ...\n")
	fmt.Fprintf(&buf, "  offsetIter: %#v\n", s.offsetIter)

	st, ok := s.iterState.(fmt.Stringer)
	if ok {
		fmt.Fprintf(&buf, "  iterState:\n%s", st.String())
	}

	return buf.String()
}
