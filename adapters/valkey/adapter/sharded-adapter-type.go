// Package adapter defines the interface for the Valkey sharded Pub/Sub adapter for Socket.IO.
package adapter

import (
	"github.com/technance-foundation/socket.io/adapters/adapter/v3"
	valkey "github.com/technance-foundation/socket.io/adapters/valkey/v3"
)

// ShardedValkeyAdapter defines the interface for a sharded Valkey-based Socket.IO adapter.
type ShardedValkeyAdapter interface {
	adapter.ClusterAdapter

	SetValkey(*valkey.ValkeyClient)
	SetOpts(any)
}
