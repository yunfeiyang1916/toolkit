package logic

import (
	"context"
	"database/sql"

	"github.com/dtm-labs/dtmcli"
	"google.golang.org/grpc/codes"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/status"

	"mall/service/product/rpc/internal/svc"
	"mall/service/product/rpc/types/product"

	"github.com/dtm-labs/dtmgrpc"
	"github.com/zeromicro/go-zero/core/logx"
)

type DecrStockLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDecrStockLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecrStockLogic {
	return &DecrStockLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DecrStock 减少产品库存
func (l *DecrStockLogic) DecrStock(in *product.DecrStockRequest) (*product.DecrStockResponse, error) {
	// 获取RawDB
	db, err := sqlx.NewMysql(l.svcCtx.Config.Mysql.DataSource).RawDB()
	if err != nil {
		return nil, status.Error(500, err.Error())
	}

	// 获取子事务屏障对象
	barrier, err := dtmgrpc.BarrierFromGrpc(l.ctx)
	if err != nil {
		return nil, status.Error(500, err.Error())
	}
	// 开启子事务屏障
	err = barrier.CallWithDB(db, func(tx *sql.Tx) error {
		// 更新产品库存
		result, err := l.svcCtx.ProductModel.TxAdjustStock(l.ctx, tx, in.Id, -1)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		// 库存不足，返回子事务失败
		if err == nil && affected == 0 {
			return dtmcli.ErrFailure
		}
		return err
	})
	// 库存不足，不在重试，走回滚
	if err == dtmcli.ErrFailure {
		return nil, status.Error(codes.Aborted, dtmcli.ResultFailure)
	}
	if err != nil {
		return nil, err
	}

	return &product.DecrStockResponse{}, nil
}
