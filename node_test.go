package storage

import (
	"context"
	"fmt"
	"testing"
)

var _ = fmt.Errorf

func TestNode(t *testing.T) {
	for _, v := range []struct {
		Keys []string
	} {
		{
			Keys: []string{
				"test/test1",
				"test/test2",
				"imgk/love",
				"imgk/you",
			},
		},
	} {
		ctx := context.Background()

		nd := new(node)
		nd.branches = make(map[string]*node)

		for _, vv := range v.Keys {
			if err := nd.Store(ctx, vv, []byte(vv)); err != nil {
				t.Errorf("store key: %v, error: %v", vv, err)
			}
		}

		for _, vv := range v.Keys {
			if bb, err := nd.Load(ctx, vv); err != nil || string(bb) != vv {
				t.Errorf("load key: %v, error: %v", vv, err)
			}
			if !nd.Exists(ctx, vv) {
				t.Errorf("exist key error")
			}
		}

		keyMap := make(map[string]struct{})
		for _, vv := range v.Keys {
			keyMap[vv] = struct{}{}
		}
		for _, v := range []string{"test/", "imgk/"} {
			keys, err := nd.List(ctx, v, false)
			if err != nil {
				t.Errorf("list key: %v, error", v)
			}
			for _, vv := range keys {
				// fmt.Println(vv)
				if _, ok := keyMap[vv]; !ok {
					t.Errorf("find key:%v, error", vv)
				}
			}
		}
	}
}
