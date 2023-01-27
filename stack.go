package syn

type stack struct {
	data []state
}

func newStack() *stack {
	return &stack{
		data: make([]state, 0),
	}
}

func (s *stack) Push(list state) {
	s.data = append(s.data, list)
}

func (s *stack) Pop(count int) {
	if len(s.data) > 0 {
		s.data = s.data[:len(s.data)-count]
	}
	return
}

func (s stack) Top() (list state) {
	if len(s.data) > 0 {
		list = s.data[len(s.data)-1]
	}
	return
}

func (s stack) Len() int {
	return len(s.data)
}

func (s stack) Clone() *stack {
	data := make([]state, len(s.data))
	copy(data, s.data)
	return &stack{
		data: data,
	}
}

func (s *stack) Clear() {
	s.data = s.data[:0]
}
