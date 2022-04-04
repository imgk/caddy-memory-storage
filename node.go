package storage

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
)

const slash = "/"

var errInvalidKey = errors.New("invalid key")

type node struct {
	mu       sync.Mutex
	keyInfo  certmagic.KeyInfo
	value    []byte
	branches map[string]*node
}

// Provision is ...
func (nd *node) Provision(ctx caddy.Context) error {
	nd.keyInfo = certmagic.KeyInfo{
		Key:        "",
		Modified:   time.Now(),
		Size:       0,
		IsTerminal: false,
	}
	nd.branches = make(map[string]*node)
	return nil
}

// Lock is ...
func (nd *node) Lock(ctx context.Context, key string) error {
	return nd.lock(ctx, strings.Split(key, slash))
}

func (nd *node) lock(ctx context.Context, keys []string) error {
	switch len(keys) {
	case 0:
		nd.mu.Lock()
		return nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			branch.mu.Lock()
			return nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.lock(ctx, keys[1:])
		}
	}

	return fs.ErrNotExist
}

// Unlock is ...
func (nd *node) Unlock(ctx context.Context, key string) error {
	return nd.unlock(ctx, strings.Split(key, slash))
}

func (nd *node) unlock(ctx context.Context, keys []string) error {
	switch len(keys) {
	case 0:
		nd.mu.Unlock()
		return nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			branch.mu.Unlock()
			return nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.unlock(ctx, keys[1:])
		}
	}

	return fs.ErrNotExist
}

// Store is ...
func (nd *node) Store(ctx context.Context, key string, value []byte) error {
	return nd.store(ctx, strings.Split(key, slash), value)
}

func (nd *node) store(ctx context.Context, keys []string, value []byte) error {
	switch len(keys) {
	case 0:
		nd.value = value
		return nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			branch.value = value
			return nil
		}
		branch := new(node)
		branch.branches = make(map[string]*node)
		branch.keyInfo.Key = keys[0]
		branch.value = value
		nd.branches[keys[0]] = branch
		return nil
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.store(ctx, keys[1:], value)
		}
		branch := new(node)
		branch.branches = make(map[string]*node)
		branch.keyInfo.Key = keys[0]
		nd.branches[keys[0]] = branch
		return branch.store(ctx, keys[1:], value)
	}

	return fs.ErrNotExist
}

// Load is ...
func (nd *node) Load(ctx context.Context, key string) ([]byte, error) {
	return nd.load(ctx, strings.Split(key, slash))
}

func (nd *node) load(ctx context.Context, keys []string) ([]byte, error) {
	switch len(keys) {
	case 0:
		return nd.value, nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.value, nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.load(ctx, keys[1:])
		}
	}

	return nil, fs.ErrNotExist
}

// Delete is ...
func (nd *node) Delete(ctx context.Context, key string) error {
	return nd.delete(ctx, strings.Split(key, slash))
}

func (nd *node) delete(ctx context.Context, keys []string) error {
	switch len(keys) {
	case 0:
		nd.value = nil
		return nil
	case 1:
		if _, ok := nd.branches[keys[0]]; ok {
			delete(nd.branches, keys[0])
			return nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.delete(ctx, keys[1:])
		}
	}

	return nil
}

// Exists is ...
func (nd *node) Exists(ctx context.Context, key string) bool {
	return nd.exist(ctx, strings.Split(key, slash))
}

func (nd *node) exist(ctx context.Context, keys []string) bool {
	switch len(keys) {
	case 0:
		return true
	case 1:
		_, ok := nd.branches[keys[0]]
		return ok
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.exist(ctx, keys[1:])
		}
	}

	return false
}

// List is ...
func (nd *node) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	return nd.list(ctx, prefix, strings.Split(strings.TrimSuffix(prefix, slash), slash), recursive)
}

func (nd *node) list(ctx context.Context, prefix string, keys []string, recursive bool) ([]string, error) {
	switch len(keys) {
	case 0:
		return []string{}, nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.dir(prefix, recursive), nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.list(ctx, prefix, keys[1:], recursive)
		}
	}

	return []string{}, fs.ErrNotExist
}

func (nd *node) dir(prefix string, recursive bool) []string {
	keys := make([]string, 0, len(nd.branches))
	for k, v := range nd.branches {
		subprefix := prefix + k
		keys = append(keys, subprefix)
		if recursive {
			keys = append(keys, v.dir(subprefix + slash, recursive)...)
		}
	}
	return keys
}

// Stat is ...
func (nd *node) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	return nd.stat(ctx, strings.Split(key, slash))
}

func (nd *node) stat(ctx context.Context, keys []string) (certmagic.KeyInfo, error) {
	switch len(keys) {
	case 0:
		return nd.keyInfo, nil
	case 1:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.keyInfo, nil
		}
	default:
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.stat(ctx, keys[1:])
		}
	}

	return certmagic.KeyInfo{}, fs.ErrNotExist
}
