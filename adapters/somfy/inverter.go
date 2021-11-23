package somfy

import "io"

type inverter struct {
	reader io.Reader
	writer io.Writer
}

func (i inverter) Read(p []byte) (int, error) {
	n, err := i.reader.Read(p)
	for i:=0;i<n;i++ {
		p[i] = ^p[i]
	}
	return n, err
}

func (i inverter) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	for i := range p {
		cp[i] = ^p[i]
	}
	n, err := i.writer.Write(cp)
	return n, err
}
