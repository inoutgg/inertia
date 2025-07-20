package sender

import (
	"context"

	"go.inout.gg/shield/shieldsender"
)

var _ shieldsender.Sender = (*Sender)(nil)

type Sender struct{}

func New() *Sender {
	return &Sender{}
}

func (s *Sender) Send(ctx context.Context, m shieldsender.Message) error {
	switch m.Key {
	// TODO: handle keys here
	}

	return nil
}
