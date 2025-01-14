package rpc

import (
	"context"
	"fmt"

	"github.com/0xPolygon/agglayer/log"
	jRPC "github.com/0xPolygon/cdk-rpc/rpc"
	"github.com/ethereum/go-ethereum/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"

	"github.com/0xPolygon/agglayer/config"
	"github.com/0xPolygon/agglayer/interop"
	"github.com/0xPolygon/agglayer/tx"
	"github.com/0xPolygon/agglayer/types"
)

// INTEROP is the namespace of the interop service
const (
	INTEROP       = "interop"
	ethTxManOwner = "interop"
	meterName     = "github.com/0xPolygon/agglayer/rpc"
)

// InteropEndpoints contains implementations for the "interop" RPC endpoints
type InteropEndpoints struct {
	executor *interop.Executor
	db       types.IDB
	config   *config.Config
	meter    metric.Meter
	logger   *zap.SugaredLogger
}

// NewInteropEndpoints returns InteropEndpoints
func NewInteropEndpoints(
	logger *zap.SugaredLogger,
	executor *interop.Executor,
	db types.IDB,
	conf *config.Config,
) *InteropEndpoints {
	meter := otel.Meter(meterName)

	return &InteropEndpoints{
		executor: executor,
		db:       db,
		config:   conf,
		meter:    meter,
		logger:   logger,
	}
}

func (i *InteropEndpoints) SendTx(signedTx tx.SignedTx) (interface{}, jRPC.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.RPC.WriteTimeout.Duration)
	defer cancel()

	i.logger.Debugf("received tx %v", signedTx.Tx)
	opts := metric.WithAttributes(attribute.Key("rollup_id").Int(int(signedTx.Tx.RollupID)))
	c, err := i.meter.Int64Counter("send_tx")
	if err != nil {
		i.logger.Warnf("failed to create send_tx counter: %s", err)
	}
	c.Add(ctx, 1, opts)

	// Check if the RPC is actually registered, if not it won't be possible to assert soundness (in the future once we are stateless won't be needed)
	if err = i.executor.CheckTx(signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("there is no RPC registered for %d", signedTx.Tx.RollupID))
	}

	// Verify ZKP using eth_call
	if err = i.executor.Verify(ctx, signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to verify tx: %s", err))
	}

	if err = i.executor.Execute(ctx, signedTx); err != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to execute tx: %s", err))
	}

	// Send L1 tx
	dbTx, err := i.db.BeginStateTransaction(ctx)
	if err != nil {
		log.Errorf("failed to begin dbTx, error: %s", err)
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, "failed to begin dbTx")
	}

	if _, err = i.executor.Settle(ctx, signedTx, dbTx); err != nil {
		if errRollback := dbTx.Rollback(ctx); errRollback != nil {
			log.Error("rollback err: ", errRollback)
		}
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to add tx to ethTxMan, error: %s", err))
	}

	if err = dbTx.Commit(ctx); err != nil {
		log.Errorf("failed to commit dbTx, error: %s", err)
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, "failed to commit dbTx")
	}

	log.Debugf("successfuly added tx %s to ethTxMan", signedTx.Tx.Hash().Hex())

	return signedTx.Tx.Hash(), nil
}

func (i *InteropEndpoints) GetTxStatus(hash common.Hash) (result interface{}, err jRPC.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), i.config.RPC.ReadTimeout.Duration)
	defer cancel()

	c, merr := i.meter.Int64Counter("get_tx_status")
	if merr != nil {
		i.logger.Warnf("failed to create get_tx_status counter: %s", merr)
	}
	c.Add(ctx, 1)

	dbTx, innerErr := i.db.BeginStateTransaction(ctx)
	if innerErr != nil {
		log.Errorf("failed to begin dbTx, error: %s", innerErr)
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, "failed to begin dbTx")
	}

	defer func() {
		if innerErr := dbTx.Rollback(ctx); innerErr != nil {
			log.Errorf("failed to rollback dbTx, error: %s", innerErr)

			result = "0x0"
			err = jRPC.NewRPCError(jRPC.DefaultErrorCode, "failed to rollback dbTx")
		}
	}()

	result, innerErr = i.executor.GetTxStatus(ctx, hash, dbTx)
	if innerErr != nil {
		return "0x0", jRPC.NewRPCError(jRPC.DefaultErrorCode, fmt.Sprintf("failed to get tx, error: %s", innerErr))
	}

	return result, nil
}
