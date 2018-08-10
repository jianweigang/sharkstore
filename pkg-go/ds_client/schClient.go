package client

import (
	"errors"
	"fmt"

	"model/pkg/metapb"
	"model/pkg/schpb"
	"util/log"

	"golang.org/x/net/context"
	"model/pkg/ds_admin"
)

// Client is a client that sends RPC.
// It should not be used after calling Close().
type SchClient interface {
	// Close should release all data.
	Close() error
	// SendKVReq sends kv request.
	CreateRange(addr string, r *metapb.Range) error
	DeleteRange(addr string, rangeId uint64) error
	TransferLeader(addr string, rangeId uint64) error
	UpdateRange(addr string, r *metapb.Range) error
	GetPeerInfo(addr string, rangeId uint64) (*schpb.GetPeerInfoResponse, error)
	SetNodeLogLevel(addr string, level string) error
	OffLineRange(addr string, rangeId uint64) error
	ReplaceRange(addr string, oldRangeId uint64, newRange *metapb.Range) error
	// ds_admin
	SetConfig(addr string, configs []*ds_adminpb.ConfigItem) error
	GetConfig(addr string, keys []*ds_adminpb.ConfigKey) (*ds_adminpb.GetConfigResponse, error)
	GetDsInfo(addr, path string) (*ds_adminpb.GetInfoResponse, error)
	ForceSplit(addr string, rangeId uint64) error
	ForceCompact(addr string, rangeId uint64) (*ds_adminpb.CompactionResponse, error)
	ClearQueue(addr string, queueType ds_adminpb.ClearQueueRequest_QueueType) (*ds_adminpb.ClearQueueResponse, error)
	GetPendingQueues(addr string, pendingType ds_adminpb.GetPendingsRequest_PendingType, count uint64) (*ds_adminpb.GetPendingsResponse, error)
	FlushDB(addr string, wait bool) error
}

type SchRpcClient struct {
	pool *ResourcePool
}

func NewSchRPCClient(opts ...int) SchClient {
	var size int
	if len(opts) == 0 {
		size = DefaultPoolSize
	} else if len(opts) > 1 {
		log.Panic("invalid client param!!!")
		return nil
	} else {
		size = opts[0]
		if size == 0 {
			log.Panic("invalid client param!!!")
			return nil
		}
	}
	return &SchRpcClient{pool: NewResourcePool(size)}
}

func (c *SchRpcClient) Close() error {
	c.pool.Close()
	return nil
}

func (c *SchRpcClient) CreateRange(addr string, r *metapb.Range) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.CreateRangeRequest{
		Header: &schpb.RequestHeader{},
		Range:  r,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.CreateRange(ctx, req)
	if err != nil {
		return err
	}
	// 目前只有这个错误，即range的元数据信息不匹配DS上已经存在的range
	// 可能是重复建立，也可能是垃圾残留
	staleRangeErr := resp.GetHeader().GetError().GetStaleRange()
	if staleRangeErr != nil {
		return fmt.Errorf("stale range %v", staleRangeErr.GetRange())
	}
	errMessage := resp.GetHeader().GetError().GetMessage()
	if len(errMessage) > 0 {
		return errors.New(errMessage)
	}
	return nil
}

func (c *SchRpcClient) DeleteRange(addr string, rangeId uint64) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.DeleteRangeRequest{
		Header:  &schpb.RequestHeader{},
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.DeleteRange(ctx, req)
	if err != nil {
		return err
	}
	staleRangeErr := resp.GetHeader().GetError().GetStaleRange()
	if staleRangeErr != nil {
		return fmt.Errorf("stale range %v", staleRangeErr.GetRange())
	}
	errMessage := resp.GetHeader().GetError().GetMessage()
	if len(errMessage) > 0 {
		return errors.New(errMessage)
	}
	return nil
}

func (c *SchRpcClient) TransferLeader(addr string, rangeId uint64) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.TransferRangeLeaderRequest{
		Header:  &schpb.RequestHeader{},
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.TransferLeader(ctx, req)
	return err
}

func (c *SchRpcClient) UpdateRange(addr string, r *metapb.Range) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.UpdateRangeRequest{
		Header: &schpb.RequestHeader{},
		Range:  r,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.UpdateRange(ctx, req)
	if err != nil {
		return err
	}
	//todo resp内部错误的判断
	return nil
}

func (c *SchRpcClient) GetPeerInfo(addr string, rangeId uint64) (*schpb.GetPeerInfoResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &schpb.GetPeerInfoRequest{
		Header:  &schpb.RequestHeader{},
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.GetPeerInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) SetNodeLogLevel(addr string, level string) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.SetNodeLogLevelRequest{
		Header: &schpb.RequestHeader{},
		Level:  level,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.SetNodeLogLevel(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) OffLineRange(addr string, rangeId uint64) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.OfflineRangeRequest{
		Header:  &schpb.RequestHeader{},
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.OfflineRange(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) ReplaceRange(addr string, oldRangeId uint64, newRange *metapb.Range) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &schpb.ReplaceRangeRequest{
		Header:     &schpb.RequestHeader{},
		OldRangeId: oldRangeId,
		NewRange:   newRange,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.ReplaceRange(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) SetConfig(addr string, configs []*ds_adminpb.ConfigItem) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &ds_adminpb.SetConfigRequest{
		Configs: configs,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.SetConfig(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) GetConfig(addr string, keys []*ds_adminpb.ConfigKey) (*ds_adminpb.GetConfigResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &ds_adminpb.GetConfigRequest{
		Key: keys,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.GetConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) GetDsInfo(addr, path string) (*ds_adminpb.GetInfoResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &ds_adminpb.GetInfoRequest{
		Path: path,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.GetDsInfo(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) ForceSplit(addr string, rangeId uint64) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &ds_adminpb.ForceSplitRequest{
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.ForceSplit(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) ForceCompact(addr string, rangeId uint64) (*ds_adminpb.CompactionResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &ds_adminpb.CompactionRequest{
		RangeId: rangeId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.ForceCompact(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) ClearQueue(addr string, queueType ds_adminpb.ClearQueueRequest_QueueType) (*ds_adminpb.ClearQueueResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &ds_adminpb.ClearQueueRequest{
		QueueType: queueType,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.ClearQueue(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) GetPendingQueues(addr string, pendingType ds_adminpb.GetPendingsRequest_PendingType, count uint64) (*ds_adminpb.GetPendingsResponse, error) {
	conn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	req := &ds_adminpb.GetPendingsRequest{
		Ptype: pendingType,
		Count: count,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	resp, err := conn.GetPendingQueues(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *SchRpcClient) FlushDB(addr string, wait bool) error {
	conn, err := c.getConn(addr)
	if err != nil {
		return err
	}
	req := &ds_adminpb.FlushDBRequest{
		Wait: wait,
	}
	ctx, cancel := context.WithTimeout(context.Background(), ReadTimeoutShort)
	defer cancel()
	_, err = conn.FlushDB(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *SchRpcClient) getConn(addr string) (RpcClient, error) {
	if len(addr) == 0 {
		return nil, errors.New("invalid address")
	}
	return c.pool.GetConn(addr)
}
