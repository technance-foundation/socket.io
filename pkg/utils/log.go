package utils

import (
	"github.com/technance-foundation/socket.io/v3/pkg/log"
)

func Log() *log.Log {
	return log.Default()
}
