package eth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/internal/ethapi"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/rpc"
)

func (api *PrivateDebugAPI) TraceBlockByNumberForZipperOne(ctx context.Context, number rpc.BlockNumber, config *TraceConfig) (map[string]interface{}, error) {
	// Fetch the block that we want to trace
	var block *types.Block

	switch number {
	case rpc.PendingBlockNumber:
		block = api.eth.miner.PendingBlock()
	case rpc.LatestBlockNumber:
		block = api.eth.blockchain.CurrentBlock()
	default:
		block = api.eth.blockchain.GetBlockByNumber(uint64(number))
	}
	// Trace the block if it was found
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return api.traceBlockForZipperOne(ctx, block)
}

func (api *PrivateDebugAPI) traceBlockForZipperOne(ctx context.Context, block *types.Block) (map[string]interface{}, error) {
	res := map[string]interface{}{
		"block":    nil,
		"receipts": nil,
		"tracer":   nil,
	}

	//GetBlockByHash
	ethapi := ethapi.NewPublicBlockChainAPI(api.eth.APIBackend)
	bres, err := ethapi.GetBlockByHash(ctx, block.Hash(), true)
	if err != nil {
		return res, err
	}
	res["block"] = bres

	//ReadReceipts
	res["receipts"] = rawdb.ReadReceipts(api.eth.ChainDb(), block.Hash(), block.Header().Number.Uint64())

	//traceBlock
	traceTimeout := "600s"
	traceTracer := "callTracer"
	tres, err := api.traceBlock(ctx, block, &TraceConfig{
		Timeout: &traceTimeout,
		Tracer:  &traceTracer,
	})
	if err != nil {
		return res, err
	}
	res["tracer"] = tres

	return res, nil
}

func (s *Ethereum) zipperone() {
	var file *os.File
	dir := "./zipperone"
	os.MkdirAll(dir, 0777)
	go func() {
		for {
			select {
			case block := <-core.ZipperoneChan:
				if s.config.SyncMode == downloader.FullSync {
					lineNum := block.Number().Int64() % 10000
					if lineNum == 0 || file == nil {
						if file != nil {
							file.Close()
						}
						f, err := os.OpenFile(fmt.Sprintf("%s/block_%05d.txt", dir, block.Number().Int64()/10000), os.O_WRONLY|os.O_CREATE, 0666)
						if err != nil {
							panic(fmt.Sprintf("%s/block_%05d.txt = %s", dir, block.Number().Int64()/10000, err))
						}
						file = f
					}
					s.zipperoneforblock(block, file, lineNum)
				}
			}
		}
	}()
}

func (s *Ethereum) zipperoneforblock(block *types.Block, file *os.File, line int64) {
	t := time.Now()
	defer func() {
		cnt := block.Transactions().Len()
		elasped := time.Now().Sub(t)
		avgElasped := int64(0)
		if cnt != 0 {
			avgElasped = elasped.Nanoseconds() / int64(cnt)
		}
		fmt.Println("ZipperOne", "Block", block.Header().Number, "tracer elasped", elasped, "txs", cnt, avgElasped, "in", file.Name())
	}()

	api := &PrivateDebugAPI{
		config: s.chainConfig,
		eth:    s,
	}

	res, err := api.traceBlockForZipperOne(context.Background(), block)
	if err != nil {
		bts, _ := json.Marshal(res)
		fmt.Println("ZipperOne", "ERR", fmt.Sprintf("trace block %s %s --- %s(%s)", block.Hash(), block.Number(), err.Error(), string(bts)))
	}

	bts, _ := json.Marshal(res)
	file.WriteString(string(bts))
	file.WriteString("\n")
}
