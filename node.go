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

var (
	errInvalidKey = errors.New("invalid key")
	errWrongType  = errors.New("invalid type")
)

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
		return errWrongType
	case 1:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			branch.mu.Lock()
			return nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
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
		return errWrongType
	case 1:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			branch.mu.Unlock()
			return nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
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
		return errWrongType
	case 1:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			if !branch.keyInfo.IsTerminal {
				return errWrongType
			}
			branch.keyInfo.Modified = time.Now()
			branch.keyInfo.Size = int64(len(value))
			branch.value = value
			return nil
		}
		branch := new(node)
		branch.keyInfo.Key = keys[0]
		branch.keyInfo.Modified = time.Now()
		branch.keyInfo.Size = int64(len(value))
		branch.keyInfo.IsTerminal = true
		branch.value = value
		nd.branches[keys[0]] = branch
		return nil
	default:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.store(ctx, keys[1:], value)
		}
		branch := new(node)
		branch.branches = make(map[string]*node)
		branch.keyInfo.Key = keys[0]
		branch.keyInfo.Modified = time.Now()
		branch.keyInfo.IsTerminal = false
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
		if nd.keyInfo.IsTerminal {
			return nil, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			if !branch.keyInfo.IsTerminal {
				return nil, errWrongType
			}
			return branch.value, nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return nil, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			if branch.keyInfo.IsTerminal {
				return nil, errWrongType
			}
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
		return errWrongType
	case 1:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
		if _, ok := nd.branches[keys[0]]; ok {
			delete(nd.branches, keys[0])
			return nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return errWrongType
		}
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
		if nd.keyInfo.IsTerminal {
			return false
		}
		_, ok := nd.branches[keys[0]]
		return ok
	default:
		if nd.keyInfo.IsTerminal {
			return false
		}
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
		return nil, nil
	case 1:
		if nd.keyInfo.IsTerminal {
			return nil, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			if nd.keyInfo.IsTerminal {
				return nil, errWrongType
			}
			return branch.dir(prefix, recursive), nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return nil, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.list(ctx, prefix, keys[1:], recursive)
		}
	}

	return nil, fs.ErrNotExist
}

func (nd *node) dir(prefix string, recursive bool) []string {
	keys := make([]string, 0, len(nd.branches))
	for k, v := range nd.branches {
		subprefix := prefix + k
		keys = append(keys, subprefix)
		if recursive {
			keys = append(keys, v.dir(subprefix+slash, recursive)...)
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
		if nd.keyInfo.IsTerminal {
			return certmagic.KeyInfo{}, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.keyInfo, nil
		}
	default:
		if nd.keyInfo.IsTerminal {
			return certmagic.KeyInfo{}, errWrongType
		}
		if branch, ok := nd.branches[keys[0]]; ok {
			return branch.stat(ctx, keys[1:])
		}
	}

	return certmagic.KeyInfo{}, fs.ErrNotExist
}
