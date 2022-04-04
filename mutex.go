package storage

import "context"

type mutex struct {
	ch chan struct{}
}

func (mu *mutex) Provision() {
	mu.ch = make(chan struct{}, 1)
}

func (mu *mutex) Lock(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case mu.ch <- struct{}{}:
	}
	return nil
}

func (mu *mutex) Unlock(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-mu.ch:
	}
	return nil
}
