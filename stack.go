package syn

type Stack struct {
	data []RuleSequence
}

func NewStack() *Stack {
	return &Stack{
		data: make([]RuleSequence, 0),
	}
}

func (s *Stack) Push(list RuleSequence) {
	s.data = append(s.data, list)
}

func (s *Stack) Pop(count int) {
	if len(s.data) > 0 {
		s.data = s.data[:len(s.data)-count]
	}
	return
}

func (s Stack) Top() (list RuleSequence) {
	if len(s.data) > 0 {
		list = s.data[len(s.data)-1]
	}
	return
}

func (s Stack) Len() int {
	return len(s.data)
}
