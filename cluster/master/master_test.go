package master

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/QuarkChain/goquarkchain/account"
	"github.com/QuarkChain/goquarkchain/cluster/config"
	"github.com/QuarkChain/goquarkchain/cluster/rpc"
	"github.com/QuarkChain/goquarkchain/cluster/service"
	qkcCommon "github.com/QuarkChain/goquarkchain/common"
	"github.com/QuarkChain/goquarkchain/consensus"
	"github.com/QuarkChain/goquarkchain/core/rawdb"
	"github.com/QuarkChain/goquarkchain/core/types"
	qrpc "github.com/QuarkChain/goquarkchain/rpc"
	"github.com/QuarkChain/goquarkchain/serialize"
	eth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/stretchr/testify/assert"
)

var (
	testGenesisTokenID = qkcCommon.TokenIDEncode("QKC")
)

type fakeRpcClient struct {
	target       string
	chainMaskLst []uint32
	slaveID      string
	chanOP       chan uint32
	config       *config.ClusterConfig
	branchs      []*account.Branch
}

func NewFakeRPCClient(chanOP chan uint32, target string, shardMaskLst []uint32, slaveID string, config *config.ClusterConfig) *fakeRpcClient {
	f := &fakeRpcClient{
		chanOP:       chanOP,
		target:       target,
		chainMaskLst: shardMaskLst,
		slaveID:      slaveID,
		config:       config,
		branchs:      make([]*account.Branch, 0),
	}
	f.initBranch()
	return f
}
func (c *fakeRpcClient) initBranch() {
	for _, v := range c.config.Quarkchain.GetGenesisShardIds() {
		if c.coverShardID(v) {
			c.branchs = append(c.branchs, &account.Branch{Value: v})
		}
	}
}
func (c *fakeRpcClient) GetOpName(op uint32) string {
	return "SB"
}

func (c *fakeRpcClient) Close() {}

func (c *fakeRpcClient) coverShardID(fullShardID uint32) bool {
	for _, chainMask := range c.chainMaskLst {
		if chainMask == fullShardID {
			return true
		}
	}
	return false

}

func (c *fakeRpcClient) Call(hostport string, req *rpc.Request) (*rpc.Response, error) {
	switch req.Op {
	case rpc.OpHeartBeat:
		if c.chanOP != nil {
			c.chanOP <- rpc.OpHeartBeat
		}
		return nil, nil
	case rpc.OpPing:
		rsp := new(rpc.Pong)
		rsp.Id = []byte(c.slaveID)
		rsp.FullShardList = c.chainMaskLst
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpConnectToSlaves:
		rsp := new(rpc.ConnectToSlavesResponse)
		rsp.ResultList = make([]*rpc.ConnectToSlavesResult, len(c.config.SlaveList))
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetUnconfirmedHeaderList:
		rsp := new(rpc.GetUnconfirmedHeadersResponse)
		for _, v := range c.branchs {
			rsp.HeadersInfoList = append(rsp.HeadersInfoList, &rpc.HeadersInfo{
				Branch:     v.Value,
				HeaderList: make([]*types.MinorBlockHeader, 0),
			})
			//rsp.HeadersInfoList[0].HeaderList = append(rsp.HeadersInfoList[0].HeaderList, &types.MinorBlockHeaderList{})
		}
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetNextBlockToMine:
		rsp := new(rpc.GetNextBlockToMineResponse)
		rsp.Block = types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetAccountData:
		rsp := new(rpc.GetAccountDataResponse)
		for _, v := range c.branchs {
			rsp.AccountBranchDataList = append(rsp.AccountBranchDataList, &rpc.AccountBranchData{Branch: v.Value})
		}
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetMine:
		return &rpc.Response{}, nil
	case rpc.OpAddRootBlock:
		rsp := new(rpc.AddRootBlockResponse)
		rsp.Switched = false
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpAddTransaction:
		return &rpc.Response{}, nil
	case rpc.OpExecuteTransaction:
		rsp := new(rpc.ExecuteTransactionResponse)
		rsp.Result = []byte("qkc")
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetMinorBlock:
		rsp := new(rpc.GetMinorBlockResponse)
		rsp.MinorBlock = types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetTransaction:
		rsp := new(rpc.GetTransactionResponse)
		rsp.MinorBlock = types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
		rsp.Index = 1
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetTransactionReceipt:
		reqData := new(rpc.GetTransactionReceiptRequest)
		err := serialize.DeserializeFromBytes(req.Data, reqData)
		if err != nil {
			panic(err)
		}
		rsp := new(rpc.GetTransactionReceiptResponse)
		rsp.MinorBlock = types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
		rsp.Index = 1
		rsp.Receipt = &types.Receipt{
			CumulativeGasUsed: 123,
		}
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetTransactionListByAddress:
		rsp := new(rpc.GetTxDetailResponse)
		rsp.Next = []byte("qkc")
		rsp.TxList = append(rsp.TxList, &rpc.TransactionDetail{
			TxHash: common.BigToHash(new(big.Int).SetUint64(11)),
		})
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetLogs:
		rsp := new(rpc.GetLogResponse)
		rsp.Logs = append(rsp.Logs, &types.Log{
			Data: []byte("qkc"),
		})
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpEstimateGas:
		rsp := new(rpc.EstimateGasResponse)
		rsp.Result = 123
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetStorageAt:
		rsp := new(rpc.GetStorageResponse)
		rsp.Result = common.BigToHash(new(big.Int).SetUint64(123))
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetCode:
		rsp := new(rpc.GetCodeResponse)
		rsp.Result = []byte("qkc")
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGasPrice:
		rsp := new(rpc.GasPriceResponse)
		rsp.Result = uint64(123)
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpMasterInfo:
		rsp := new(rpc.MasterInfo)
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpGetWork:
		rsp := new(consensus.MiningWork)
		rsp.Number = 1
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	case rpc.OpSubmitWork:
		rsp := new(rpc.SubmitWorkResponse)
		rsp.Success = true
		data, err := serialize.SerializeToBytes(rsp)
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Data: data}, nil
	default:
		return nil, errors.New("unkown code")
	}
}

func initEnv(t *testing.T, chanOp chan uint32) *QKCMasterBackend {
	return initEnvWithConsensusType(t, chanOp, config.PoWSimulate, "")
}

func initEnvWithConsensusType(t *testing.T, chanOp chan uint32, consensusType string, pubKey string) *QKCMasterBackend {
	monkey.Patch(NewSlaveConn, func(target string, shardMaskLst []uint32, slaveID string) *SlaveConnection {
		client := NewFakeRPCClient(chanOp, target, shardMaskLst, slaveID, config.NewClusterConfig())
		return &SlaveConnection{
			target:        target,
			client:        client,
			shardMaskList: shardMaskLst,
			slaveID:       slaveID,
		}
	})
	monkey.Patch(createDB, func(ctx *service.ServiceContext, name string, clean bool, isReadOnly bool) (ethdb.Database, error) {
		return service.NewQkcMemoryDB(isReadOnly), nil
	})
	defer monkey.UnpatchAll()

	ctx := &service.ServiceContext{}
	clusterConfig := config.NewClusterConfig()
	clusterConfig.Quarkchain.Root.ConsensusType = consensusType
	clusterConfig.Quarkchain.Root.ConsensusConfig.RemoteMine = true
	clusterConfig.Quarkchain.Root.Genesis.Difficulty = 2000
	clusterConfig.Quarkchain.GuardianPublicKey = common.FromHex(pubKey)
	master, err := New(ctx, clusterConfig)
	if err != nil {
		panic(err)
	}

	if err := master.Init(nil); err != nil {
		assert.NoError(t, err)
	}
	return master
}
func TestMasterBackend_InitCluster(t *testing.T) {
	initEnv(t, nil)
}

func TestMasterBackend_HeartBeat(t *testing.T) {
	chanOp := make(chan uint32, 100)
	master := initEnv(t, chanOp)
	master.Heartbeat()
	status := true
	countHeartBeat := 0
	for status {
		select {
		case op := <-chanOp:
			if op == rpc.OpHeartBeat {
				countHeartBeat++
			}
			if countHeartBeat == len(master.clusterConfig.SlaveList) {
				status = false
			}
		case <-time.After(2 * time.Second):
			panic(errors.New("no receive Heartbeat"))
		}
	}
}

func TestGetSlaveConnByBranch(t *testing.T) {
	master := initEnv(t, nil)
	for _, v := range master.clusterConfig.Quarkchain.GetGenesisShardIds() {
		conn := master.GetOneSlaveConnById(v)
		assert.NotNil(t, conn)
	}
	fakeFullShardID := uint32(99999)
	conn := master.GetOneSlaveConnById(fakeFullShardID)
	assert.Nil(t, conn)
}

func TestCreateRootBlockToMine(t *testing.T) {
	minorBlock := types.NewMinorBlock(&types.MinorBlockHeader{}, &types.MinorBlockMeta{}, nil, nil, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	master := initEnv(t, nil)
	rawdb.WriteMinorBlock(master.chainDb, minorBlock)
	rootBlock, err := master.createRootBlockToMine(add1)
	assert.NoError(t, err)
	assert.Equal(t, rootBlock.Header().Signature, [65]byte{})
	assert.Equal(t, rootBlock.Header().Coinbase, add1)
	assert.Equal(t, rootBlock.CoinbaseAmount().GetTokenBalance(testGenesisTokenID).String(), "120000000000000000000")
	assert.Equal(t, rootBlock.Header().Difficulty, new(big.Int).SetUint64(2000))

	rawdb.DeleteBlock(master.chainDb, minorBlock.Hash())
	rootBlock, err = master.createRootBlockToMine(add1)
	assert.NoError(t, err)
	assert.Equal(t, rootBlock.Header().Coinbase, add1)
	assert.Equal(t, rootBlock.CoinbaseAmount().GetTokenBalance(testGenesisTokenID).String(), "120000000000000000000")
	assert.Equal(t, rootBlock.Header().Difficulty, new(big.Int).SetUint64(2000))
	assert.Equal(t, len(rootBlock.MinorBlockHeaders()), 0)
}

func TestCreateRootBlockToMineWithSign(t *testing.T) {
	minorBlock := types.NewMinorBlock(&types.MinorBlockHeader{}, &types.MinorBlockMeta{}, nil, nil, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	master := initEnv(t, nil)
	key, err := crypto.ToECDSA(id1.GetKey().Bytes())
	assert.NoError(t, err)
	master.clusterConfig.Quarkchain.RootSignerPrivateKey = id1.GetKey().Bytes()
	master.clusterConfig.Quarkchain.GuardianPublicKey = crypto.FromECDSAPub(&key.PublicKey)
	rawdb.WriteMinorBlock(master.chainDb, minorBlock)
	rootBlock, err := master.createRootBlockToMine(add1)
	assert.NoError(t, err)
	assert.NotEqual(t, rootBlock.Header().Signature, [65]byte{})
	assert.Equal(t, rootBlock.Header().Coinbase, add1)
	assert.Equal(t, rootBlock.CoinbaseAmount().GetTokenBalance(master.clusterConfig.Quarkchain.GetDefaultChainTokenID()).String(), "120000000000000000000")
	assert.Equal(t, rootBlock.Header().Difficulty, new(big.Int).SetUint64(2000))
}

func TestGetAccountData(t *testing.T) {
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	master := initEnv(t, nil)
	_, err = master.GetAccountData(&add1, nil)
	assert.NoError(t, err)
}

func TestGetPrimaryAccountData(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	_, err = master.GetPrimaryAccountData(&add1, nil)
	assert.NoError(t, err)
}

func TestSendMiningConfigToSlaves(t *testing.T) {
	master := initEnv(t, nil)
	err := master.SendMiningConfigToSlaves(true)
	assert.NoError(t, err)
}

func TestAddRootBlock(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	rootBlock, err := master.rootBlockChain.CreateBlockToMine(nil, &add1, nil)
	assert.NoError(t, err)
	err = master.AddRootBlock(rootBlock)
	assert.NoError(t, err)
}

func TestSetTargetBlockTime(t *testing.T) {
	master := initEnv(t, nil)
	rootBlockTime := uint32(12)
	minorBlockTime := uint32(1)
	err := master.SetTargetBlockTime(&rootBlockTime, &minorBlockTime)
	assert.NoError(t, err)
}

func TestAddTransaction(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	evmTx := types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int).SetUint64(10000000), 2, 2, 1, 0, []byte{}, 0, 0)
	tx := &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	err = master.AddTransaction(tx)
	assert.NoError(t, err)

	evmTx = types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int).SetUint64(1000000000), 2, 2, 1, 0, []byte{}, 0, 0)
	tx = &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	err = master.AddTransaction(tx)
	assert.NoError(t, err)

	//fromFullShardKey 00040000 -> chainID =4
	// config->chainID : 1,2,3
	evmTx = types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int).SetUint64(1000000000), 262144, 2, 1, 0, []byte{}, 0, 0)
	tx = &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	err = master.AddTransaction(tx)
	assert.Error(t, err)
}

func TestExecuteTransaction(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	evmTx := types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 2, 2, 1, 0, []byte{}, 0, 0)
	tx := &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	data, err := master.ExecuteTransaction(tx, &add1, nil)
	assert.NoError(t, err)
	assert.Equal(t, data, []byte("qkc"))

	evmTx = types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 222222222, 2, 1, 0, []byte{}, 0, 0)
	tx = &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	_, err = master.ExecuteTransaction(tx, &add1, nil)
	assert.Error(t, err)
}

func TestGetMinorBlockByHeight(t *testing.T) {
	master := initEnv(t, nil)
	fakeMinorBlock := types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
	fakeShardStatus := rpc.ShardStatus{
		Branch: account.Branch{Value: 2},
		Height: 0,
	}
	master.UpdateShardStatus(&fakeShardStatus)
	minorBlock, _, err := master.GetMinorBlockByHeight(nil, account.Branch{Value: 2}, false)

	assert.NoError(t, err)
	assert.Equal(t, fakeMinorBlock.Hash(), minorBlock.Hash())

	_, _, err = master.GetMinorBlockByHeight(nil, account.Branch{Value: 2222}, false)
	assert.Error(t, err)
}

func TestGetMinorBlockByHash(t *testing.T) {
	master := initEnv(t, nil)
	fakeMinorBlock := types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
	minorBlock, _, err := master.GetMinorBlockByHash(common.Hash{}, account.Branch{Value: 2}, false)
	assert.NoError(t, err)
	assert.Equal(t, fakeMinorBlock.Hash(), minorBlock.Hash())

	_, _, err = master.GetMinorBlockByHash(common.Hash{}, account.Branch{Value: 2222}, false)
	assert.Error(t, err)
}

func TestGetTransactionByHash(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	evmTx := types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 2, 2, 1, 0, []byte{}, 0, 0)
	tx := &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	fakeMinorBlock := types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
	minorBlock, index, err := master.GetTransactionByHash(tx.Hash(), account.Branch{Value: 2})
	assert.NoError(t, err)
	assert.Equal(t, index, uint32(1))
	assert.Equal(t, fakeMinorBlock.Hash(), minorBlock.Hash())
}

func TestGetTransactionReceipt(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	evmTx := types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 2, 2, 1, 0, []byte{}, 0, 0)
	tx := &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	fakeMinorBlock := types.NewMinorBlock(&types.MinorBlockHeader{Version: 111}, &types.MinorBlockMeta{}, nil, nil, nil)
	MinorBlock, _, rep, err := master.GetTransactionReceipt(tx.Hash(), account.Branch{Value: 2})
	assert.NoError(t, err)
	assert.Equal(t, MinorBlock.Hash(), fakeMinorBlock.Hash())
	assert.Equal(t, rep.CumulativeGasUsed, uint64(123))
}

func TestGetTransactionsByAddress(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	res, bytes, err := master.GetTransactionsByAddress(&add1, []byte{}, 0, nil)
	assert.NoError(t, err)
	assert.Equal(t, bytes, []byte("qkc"))
	assert.Equal(t, res[0].TxHash, common.BigToHash(new(big.Int).SetUint64(11)))
}

func TestGetLogs(t *testing.T) {
	master := initEnv(t, nil)

	startBlock := qrpc.BlockNumber(0)
	endBlock := qrpc.BlockNumber(0)
	logs, err := master.GetLogs(&qrpc.FilterQuery{
		FullShardId: 2,
		FilterQuery: eth.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
		}})
	assert.NoError(t, err)
	assert.Equal(t, len(logs), 1)
	assert.Equal(t, logs[0].Data, []byte("qkc"))
}

func TestEstimateGas(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	evmTx := types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 2, 2, 1, 0, []byte{}, 0, 0)
	tx := &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	data, err := master.EstimateGas(tx, &add1)
	assert.NoError(t, err)
	if !tx.EvmTx.IsCrossShard() {
		assert.Equal(t, data, uint32(123))
	} else {
		assert.Equal(t, data, uint32(123+9000))
	}

	evmTx = types.NewEvmTransaction(0, id1.GetRecipient(), new(big.Int), 0, new(big.Int), 2222222, 2, 1, 0, []byte{}, 0, 0)
	tx = &types.Transaction{
		EvmTx:  evmTx,
		TxType: types.EvmTx,
	}
	data, err = master.EstimateGas(tx, &add1)
	assert.Error(t, err)
}

func TestGetStorageAt(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	data, err := master.GetStorageAt(&add1, common.Hash{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, data.Big().Uint64(), uint64(123))
}

func TestGetCode(t *testing.T) {
	master := initEnv(t, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	data, err := master.GetCode(&add1, nil)
	assert.NoError(t, err)
	assert.Equal(t, data, []byte("qkc"))

}

func TestGasPrice(t *testing.T) {
	master := initEnv(t, nil)
	data, err := master.GasPrice(account.Branch{Value: 2}, testGenesisTokenID)
	assert.NoError(t, err)
	assert.Equal(t, data, uint64(123))
}

func TestGetWork(t *testing.T) {
	master := initEnv(t, nil)
	var id uint32 = 2
	data, err := master.GetWork(&id, nil)
	assert.NoError(t, err)
	assert.Equal(t, data.Number, uint64(1))
}

func TestSubmitWork(t *testing.T) {
	master := initEnv(t, nil)
	var id uint32 = 2
	data, err := master.SubmitWork(&id, common.Hash{}, 0, common.Hash{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, data, true)
}

func TestSubmitWorkForRootChain(t *testing.T) {
	minorBlock := types.NewMinorBlock(&types.MinorBlockHeader{}, &types.MinorBlockMeta{}, nil, nil, nil)
	id1, err := account.CreatRandomIdentity()
	assert.NoError(t, err)
	add1 := account.NewAddress(id1.GetRecipient(), 3)
	key, err := crypto.ToECDSA(id1.GetKey().Bytes())
	assert.NoError(t, err)
	master := initEnvWithConsensusType(t, nil, config.PoWDoubleSha256, common.ToHex(crypto.FromECDSAPub(&key.PublicKey))) //common.Bytes2Hex(key.PublicKey.X.Bytes())+common.Bytes2Hex(key.PublicKey.Y.Bytes())
	master.miner.SetMining(true)
	rawdb.WriteMinorBlock(master.chainDb, minorBlock)
	rootBlock, err := master.createRootBlockToMine(add1)
	assert.NoError(t, err)
	results := make(chan<- types.IBlock, 10)
	err = master.engine.Seal(master.rootBlockChain, rootBlock, rootBlock.Difficulty(), 1, results, nil)
	assert.NoError(t, err)
	sig, err := crypto.Sign(rootBlock.Header().SealHash().Bytes(), key)
	assert.NoError(t, err)
	assert.Equal(t, len(sig), 65)
	signature := [65]byte{}
	copy(signature[:], sig[:])
	nonce := findNonce(master.engine, rootBlock.Header(),
		new(big.Int).Div(rootBlock.Difficulty(), new(big.Int).SetUint64(1000)))
	data, err := master.SubmitWork(nil, rootBlock.Header().SealHash(), nonce, common.Hash{}, &signature)
	assert.NoError(t, err)
	assert.Equal(t, true, data)
}

func findNonce(engine consensus.Engine, header *types.RootBlockHeader, difficalty *big.Int) uint64 {
	for {
		if err := engine.VerifySeal(nil, header, difficalty); err == nil {
			return header.Nonce
		}
		header.Nonce = header.Nonce + 1
	}
}
