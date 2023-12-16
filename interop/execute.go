package interop

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xPolygon/beethoven/tx"
	"github.com/jackc/pgx/v4"

	"github.com/0xPolygon/cdk-validium-node/jsonrpc/client"
	"github.com/0xPolygon/cdk-validium-node/log"
	"github.com/ethereum/go-ethereum/common"
)

const ethTxManOwner = "interop"

func (e *Executor) Execute(signedTx tx.SignedTx) error {
	ctx := context.TODO()

	// Check expected root vs root from the managed full node
	// TODO: go stateless, depends on https://github.com/0xPolygonHermez/zkevm-prover/issues/581
	// when this happens we should go async from here, since processing all the batches could take a lot of time
	zkEVMClient := client.NewClient(e.config.FullNodeRPCs[signedTx.Tx.L1Contract])
	batch, err := zkEVMClient.BatchByNumber(
		ctx,
		big.NewInt(int64(signedTx.Tx.NewVerifiedBatch)),
	)
	if err != nil {
		return fmt.Errorf("failed to get batch from our node, error: %s", err)
	}
	if batch.StateRoot != signedTx.Tx.ZKP.NewStateRoot || batch.LocalExitRoot != signedTx.Tx.ZKP.NewLocalExitRoot {
		return fmt.Errorf(
			"Missmatch detected,  expected local exit root: %s actual: %s. expected state root: %s actual: %s",
			signedTx.Tx.ZKP.NewLocalExitRoot.Hex(),
			batch.LocalExitRoot.Hex(),
			signedTx.Tx.ZKP.NewStateRoot.Hex(),
			batch.StateRoot.Hex(),
		)
	}

	return nil
}

func (e *Executor) Settle(signedTx tx.SignedTx, dbTx pgx.Tx) (common.Hash, error) {
	// // Send L1 tx
	// Verify ZKP using eth_call
	l1TxData, err := e.etherman.BuildTrustedVerifyBatchesTxData(
		uint64(signedTx.Tx.LastVerifiedBatch),
		uint64(signedTx.Tx.NewVerifiedBatch),
		signedTx.Tx.ZKP,
	)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to build verify ZKP tx: %s", err)
	}

	if err := e.ethTxMan.Add(
		context.Background(),
		ethTxManOwner,
		signedTx.Tx.Hash().Hex(),
		e.interopAdminAddr,
		&signedTx.Tx.L1Contract,
		nil,
		l1TxData,
		dbTx,
	); err != nil {
		return common.Hash{}, fmt.Errorf("failed to add tx to ethTxMan, error: %s", err)
	}
	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())
	return signedTx.Tx.Hash(), nil
}