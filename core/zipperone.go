package core

import (
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ZipperoneChan = make(chan *types.Block, 100)
)
