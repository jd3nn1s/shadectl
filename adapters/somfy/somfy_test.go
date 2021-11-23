package somfy

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSomfy_SetPosition(t *testing.T) {
	b := bytes.Buffer{}

	s := &somfy{
		motorAddress: [3]byte{0xae, 0x85, 0x0c},
		writer: &b,
	}

	s.SetPosition(5)
	log.Printf("msg bytes: %x", b.Bytes())
}

func TestSomfy_GetPosition(t *testing.T) {
	b := bytes.Buffer{}
	reader := bytes.NewBuffer([]byte{0xF2, 0xEF, 0x8F, 0x51, 0x7A, 0xF3, 0x80, 0x80, 0x80, 0x9B, 0xFF, 0xE9, 0x00, 0xFF, 0x08, 0x31})

	s := &somfy{
		motorAddress: [3]byte{0xae, 0x85, 0x0c},
		writer: &b,
		reader: reader,
	}

	pos, err := s.GetPosition()
	assert.NoError(t, err)

	log.Printf("percentage: %x", pos)
	log.Printf("msg bytes: %x", b.Bytes())

}
