package syn

type Stack struct {
	data []State
}

func NewStack() *Stack {
	return &Stack{
		data: make([]State, 0),
	}
}

func (s *Stack) Push(list State) {
	s.data = append(s.data, list)
}

func (s *Stack) Pop(count int) {
	if len(s.data) > 0 {
		s.data = s.data[:len(s.data)-count]
	}
	return
}

func (s Stack) Top() (list State) {
	if len(s.data) > 0 {
		list = s.data[len(s.data)-1]
	}
	return
}

func (s Stack) Len() int {
	return len(s.data)
}
