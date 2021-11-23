package inmem

type inmem struct {
	position int
}

func New() *inmem {
	return &inmem{}
}

func (b *inmem) SetPosition(position int) error {
	b.position = position
	return nil
}

func (b *inmem) GetPosition() (int, error) {
	return b.position, nil
}
