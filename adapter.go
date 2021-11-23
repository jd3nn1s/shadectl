package shadectl

type ShadeAdapter interface {
	SetPosition(position int) error
	GetPosition() (int, error)
}

type Service interface {
	SetPosition(position int) error
	GetPosition() (int, error)
}

type Svc struct {
	shadeAdapter ShadeAdapter
}

func NewService(blindAdapter ShadeAdapter) Service {
	return &Svc{
		shadeAdapter: blindAdapter,
	}
}

func (s *Svc) SetPosition(position int) error {
	return s.shadeAdapter.SetPosition(position)
}

func (s *Svc) GetPosition() (int, error) {
	return s.shadeAdapter.GetPosition()
}