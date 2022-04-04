package storage

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/certmagic"
)

func init() {
	caddy.RegisterModule(StorageConverter{})
}

// StorageConverter is ...
type StorageConverter struct {
	node
}

// CaddyModule is ...
func (StorageConverter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.storage.memory",
		New: func() caddy.Module { return new(StorageConverter) },
	}
}

// CertMagicStorage is ...
func (sc *StorageConverter) CertMagicStorage() (certmagic.Storage, error) {
	return &sc.node, nil
}

var (
	_ caddy.StorageConverter = (*StorageConverter)(nil)
	_ caddy.Provisioner      = (*StorageConverter)(nil)
)
