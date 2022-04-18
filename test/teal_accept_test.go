package test

import (
	"context"
	"testing"
	"os"
	"flag"
	"fmt"
	"encoding/base64"
	"encoding/binary"
	"strconv"

	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/stretchr/testify/require"

	//"encoding/hex"
)

var (
	algodHost = flag.String("algod_host", "http://localhost:4001", "Host of algod client")
	algodTokenPath = flag.String("algod_token_path", "../algorand/algod.token", "Path to algod token")
	kmdHost = flag.String("kmd_host", "http://localhost:4002", "Host of kmd client")
	kmdTokenPath = flag.String("kmd_token_path", "../algorand/kmd.token", "Path to kmd token")
	indexerHost = flag.String("indexer_host", "http://localhost:4003", "Host of indexer client")
)

func TestSome(t *testing.T) {

	// abcdef
	// - 6*6 bits = 36 bits
	// - 4.5 bytes
	//
	// 011010
	// 011011
	// 011100
	// 011101
	// 011110
	// 011111

	// 0110 1001 1011 0111 0001 1101 0111 1001 1111
	// 6    9    b    7    1    d    7    9    f

	raw := b64StrToBytes(t, "abcdefA=")

	std, _ := base64.StdEncoding.DecodeString("abcdefA=")

	fmt.Printf("raw: %x\n", raw)
	fmt.Printf("std: %x\n", std)

}

func TestContractCommitDraw(t *testing.T) {

	creator := crypto.GenerateAccount()
	acc1 := crypto.GenerateAccount()

	appID := fundAccountsAndDeployContract(t, creator, acc1)

	fmt.Println("Creator:")
	fmt.Println("addr:", creator.Address.String())
	fmt.Println("pub key:", base64.StdEncoding.EncodeToString(creator.PublicKey))
	fmt.Println("priv key:", base64.StdEncoding.EncodeToString(creator.PrivateKey))

	testCases := []struct{
		Name string
		Txs []TxCreator
		Sender crypto.Account
		ExpectTxBroadcastError bool
		ExpectedLocalState map[string]string
		ExpectedGlobalState map[string]string
	}{
		{
			Name: "inital global state",
			ExpectedGlobalState: map[string]string{
				"owner": base64.StdEncoding.EncodeToString(creator.PublicKey),
				"numTickets": "0",
				"draw": "",
				"1p": "0",
				"1s": "0",
				"2p": "0",
				"2s": "0",
				"3p": "0",
				"3s": "0",
				"4p": "0",
				"4s": "0",
				"5p": "0",
				"5s": "0",
				"6p": "0",
				"6s": "0",
			},
		},
		{
			Name: "opt-in to app",
			Txs: []TxCreator{
				TxAppOptIn{
					AppID: appID,
					Sender: acc1,
				},
			},
			Sender: acc1,
			ExpectedLocalState: map[string]string{
				"wager": "0",
				"commitment": "",
			},
			ExpectedGlobalState: map[string]string{
				"owner": base64.StdEncoding.EncodeToString(creator.PublicKey),
				"numTickets": "0",
				"draw": "",
				"1p": "0",
				"1s": "0",
				"2p": "0",
				"2s": "0",
				"3p": "0",
				"3s": "0",
				"4p": "0",
				"4s": "0",
				"5p": "0",
				"5s": "0",
				"6p": "0",
				"6s": "0",
			},
		},
		{
			Name: "calling commit without payment tx throws broadcast error",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc1,
					Method: "Commit",
					Args: [][]byte{
						[]byte("abcdef"),
					},
				},
			},
			Sender: acc1,
			ExpectTxBroadcastError: true,
		},
		{
			Name: "calling commit with payment tx succeeds",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc1,
					Method: "Commit",
					Args: [][]byte{
						b64StrToBytes(t, "abcdefA="),
					},
				},
				TxPayment{
					From: acc1,
					To: crypto.GetApplicationAddress(appID),
					Amount: 1000000,
				},
			},
			Sender: acc1,
			ExpectedLocalState: map[string]string{
				"wager": "1000000",
				"commitment": "abcdefA=",
			},
			ExpectedGlobalState: map[string]string{
				"owner": base64.StdEncoding.EncodeToString(creator.PublicKey),
				"numTickets": "1",
				"draw": "",
				"1p": "0",
				"1s": "0",
				"2p": "0",
				"2s": "0",
				"3p": "0",
				"3s": "0",
				"4p": "0",
				"4s": "0",
				"5p": "0",
				"5s": "0",
				"6p": "0",
				"6s": "0",
			},
		},
		{
			Name: "non-creator calls draw fails to broadcast",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc1,
					Method: "SetDraw",
					Args: [][]byte{
						b64StrToBytes(t, "abcdefA="),
						uint64ToBytes(t, 6), // 1s
						uint64ToBytes(t, 61),
						uint64ToBytes(t, 5), // 2s
						uint64ToBytes(t, 51),
						uint64ToBytes(t, 4), // 3s
						uint64ToBytes(t, 41),
						uint64ToBytes(t, 3), // 4s
						uint64ToBytes(t, 31),
						uint64ToBytes(t, 2), // 5s
						uint64ToBytes(t, 21),
						uint64ToBytes(t, 1), // 6s
						uint64ToBytes(t, 500000),
					},
				},
			},
			Sender: acc1,
			ExpectTxBroadcastError: true,
		},
		{
			Name: "creator sets draw succeeds",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: creator,
					Method: "SetDraw",
					Args: [][]byte{
						b64StrToBytes(t, "abcdefA="),
						uint64ToBytes(t, 6), // 1s
						uint64ToBytes(t, 61),
						uint64ToBytes(t, 5), // 2s
						uint64ToBytes(t, 51),
						uint64ToBytes(t, 4), // 3s
						uint64ToBytes(t, 41),
						uint64ToBytes(t, 3), // 4s
						uint64ToBytes(t, 31),
						uint64ToBytes(t, 2), // 5s
						uint64ToBytes(t, 21),
						uint64ToBytes(t, 1), // 6s
						uint64ToBytes(t, 500000),
					},
				},
			},
			Sender: creator,
			ExpectedGlobalState: map[string]string{
				"owner": base64.StdEncoding.EncodeToString(creator.PublicKey),
				"numTickets": "1",
				"draw": "abcdefA=",
				"1s": "6",
				"1p": "61",
				"2s": "5",
				"2p": "51",
				"3s": "4",
				"3p": "41",
				"4s": "3",
				"4p": "31",
				"5s": "2",
				"5p": "21",
				"6s": "1",
				"6p": "500000",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			if test.ExpectTxBroadcastError {
				requireTxBroadcastError(t, test.Txs...)
				return
			}

			if len(test.Txs) > 0 {
				broadcastTxsAndWait(t, test.Txs...)
			}

			if len(test.ExpectedLocalState) > 0 {
				localState := getAppLocalState(t, appID, test.Sender.Address)
				require.Equal(t, test.ExpectedLocalState, localState)
			}

			if len(test.ExpectedGlobalState) > 0 {
				globalState := getAppGlobalState(t, appID)
				require.Equal(t, test.ExpectedGlobalState, globalState)
			}
		})
	}
}

func fundAccountsAndDeployContract(t *testing.T, creator crypto.Account, accounts ...crypto.Account) uint64 {

	kmdAcc := getKMDAccount(t)

	var txs []TxCreator
	txs = append(txs, TxPayment{
		From: kmdAcc,
		To: creator.Address,
		Amount: 2000000,
	})

	for _, acc := range accounts {
		txs = append(txs, TxPayment{
			From: kmdAcc,
			To: acc.Address,
			Amount: 2000000,
		})
	}

	txs = append(txs, TxAppDeploy{
		Creator: creator,
		ApprovalPath: "../contract/approval.teal",
		ClearPath: "../contract/clear.teal",
		GlobalUints: 13,
		GlobalByteSlices: 2,
		LocalUints: 1,
		LocalByteSlices:1,
	})

	txIDs := broadcastTxsAndWait(t, txs...)

	algodCl := algodClient(t)
	pendingRes, _, err := algodCl.PendingTransactionInformation(txIDs[len(txIDs)-1]).Do(context.Background())
	require.NoError(t, err)

	return pendingRes.ApplicationIndex
}


func TestPokeExisting(t *testing.T) {

	addr, err := types.DecodeAddress("OCTAWZ77S7VISVSQREH5MONA4J5BU7SNHEL74XHVUGWNYC6U4NQGWANWTE")
	require.NoError(t, err)
	pubk, err := base64.StdEncoding.DecodeString("cKYLZ/+X6olWUIkP1jmg4noafk05F/5c9aGs3AvU42A=")
	require.NoError(t, err)
	privk, err := base64.StdEncoding.DecodeString("YPlwraDB/vC+XVtPmmXYydkzvJAbgr+wkGJ/TtntZidwpgtn/5fqiVZQiQ/WOaDiehp+TTkX/lz1oazcC9TjYA==")
	require.NoError(t, err)

	acc := crypto.Account{
		Address: addr,
		PublicKey: pubk,
		PrivateKey: privk,
	}

	appID := uint64(86)

	fmt.Println("app addr:", crypto.GetApplicationAddress(appID))

	broadcastTxsAndWait(
		t,
		TxAppCall{
			AppID: appID,
			Sender: acc,
			Method: "SetDraw",
			Args: [][]byte{
				b64StrToBytes(t, "abcdefA="),
				uint64ToBytes(t, 6), // 1s
				//uint64ToBytes(t, 61),
				//uint64ToBytes(t, 5), // 2s
				//uint64ToBytes(t, 51),
				//uint64ToBytes(t, 4), // 3s
				//uint64ToBytes(t, 41),
				//uint64ToBytes(t, 3), // 4s
				//uint64ToBytes(t, 31),
				//uint64ToBytes(t, 2), // 5s
				//uint64ToBytes(t, 21),
				//uint64ToBytes(t, 1), // 6s
				//uint64ToBytes(t, 500000),
			},
		},
	)

	require.Equal(
		t,
		map[string]string{
			"owner": base64.StdEncoding.EncodeToString(acc.PublicKey),
			"numTickets": "1",
			"draw": "abcdefA=",
			"1s": "6",
			"1p": "61",
			"2s": "5",
			"2p": "51",
			"3s": "4",
			"3p": "41",
			"4s": "3",
			"4p": "31",
			"5s": "2",
			"5p": "21",
			"6s": "1",
			"6p": "500000",
		},
		getAppGlobalState(t, appID),
	)
}

func algodClient(t *testing.T) *algod.Client {

	token, err := os.ReadFile(*algodTokenPath)
	require.NoError(t, err)

	c, err := algod.MakeClient(*algodHost, string(token))
	require.NoError(t, err)

	return c
}

func kmdClient(t *testing.T) kmd.Client {

	token, err := os.ReadFile(*kmdTokenPath)
	require.NoError(t, err)

	c, err := kmd.MakeClient(*kmdHost, string(token))
	require.NoError(t, err)

	return c
}

func indexerClient(t *testing.T) *indexer.Client {

	// TODO: Figure out if we need to pass a token to the indexer client here
	c, err := indexer.MakeClient(*indexerHost, "")
	require.NoError(t, err)

	return c
}

// getKMDAccount returns the Account (i.e. Priv+Pub keys) of the first account in KMD
func getKMDAccount(t *testing.T) crypto.Account {

	kmd := kmdClient(t)
	w, err := kmd.ListWallets()
	require.NoError(t, err)
	require.Greater(t, len(w.Wallets), 0)

	walletPassword := ""
	handleResp, err := kmd.InitWalletHandle(w.Wallets[0].ID, walletPassword)
	require.NoError(t, err)
	handle := handleResp.WalletHandleToken

	keys, err := kmd.ListKeys(handle)
	require.NoError(t, err)
	require.Greater(t, len(keys.Addresses), 0)

	res, err := kmd.ExportKey(handle, walletPassword, keys.Addresses[0])
	require.NoError(t, err)
	kmdAccount, err := crypto.AccountFromPrivateKey(res.PrivateKey)
	require.NoError(t, err)

	return kmdAccount
}

func compileTeal(t *testing.T, tealPath string) []byte {

	srcBytes, err := os.ReadFile(tealPath)
	require.NoError(t, err)
	algod := algodClient(t)
	res, err := algod.TealCompile(srcBytes).Do(context.Background())
	require.NoError(t, err)
	prog, err := base64.StdEncoding.DecodeString(res.Result)
	require.NoError(t, err)
	return prog
}

type TxCreator interface {
	Create(t *testing.T) (future.TransactionWithSigner)
}

type TxAppDeploy struct {
	Creator crypto.Account
	ApprovalPath string
	ClearPath string
	GlobalUints uint64
	GlobalByteSlices uint64
	LocalUints uint64
	LocalByteSlices uint64
}

func (c TxAppDeploy) Create(t *testing.T) (future.TransactionWithSigner) {

	appCreate := TxAppCreate{
		ApprovalProg: compileTeal(t, c.ApprovalPath),
		ClearProg: compileTeal(t, c.ClearPath),
		GlobalUints: c.GlobalUints,
		GlobalByteSlices: c.GlobalByteSlices,
		LocalUints: c.LocalUints,
		LocalByteSlices: c.LocalByteSlices,
		Creator: c.Creator,
	}

	return appCreate.Create(t)
}

type TxAppCreate struct {
	Creator crypto.Account // Tx sender will be the signer of this tx
	OptIn bool
	ApprovalProg []byte
	ClearProg []byte
	GlobalUints uint64
	GlobalByteSlices uint64
	LocalUints uint64
	LocalByteSlices uint64
	AppArgs [][]byte
	Accounts []string
	ForeignApps []uint64
	ForeignAssets []uint64
	Note []byte
	Group types.Digest
	Lease [32]byte
	RekeyTo types.Address
}

func (c TxAppCreate) Create(t *testing.T) (future.TransactionWithSigner) {

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	tx, err := future.MakeApplicationCreateTx(
		c.OptIn,
		c.ApprovalProg,
		c.ClearProg,
		types.StateSchema{
			NumUint: c.GlobalUints,
			NumByteSlice: c.GlobalByteSlices,
		},
		types.StateSchema{
			NumUint: c.LocalUints,
			NumByteSlice: c.LocalByteSlices,
		},
		c.AppArgs,
		c.Accounts,
		c.ForeignApps,
		c.ForeignAssets,
		txParams,
		c.Creator.Address,
		c.Note,
		c.Group,
		c.Lease,
		c.RekeyTo,
	)
	require.NoError(t, err)

	return future.TransactionWithSigner{
		Txn: tx,
		Signer: future.BasicAccountTransactionSigner{
			Account: c.Creator,
		},
	}
}

type TxAppOptIn struct {
	AppID uint64
	Args []string
	Accounts []string
	ForeignApps []uint64
	ForeignAssets []uint64
	Note []byte
	Group types.Digest
	Lease [32]byte
	RekeyTo types.Address
	Sender crypto.Account // Tx sender will be the signer of this tx
}

func (c TxAppOptIn) Create(t *testing.T) (future.TransactionWithSigner) {

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	var appArgs [][]byte
	for _, arg := range c.Args {
		appArgs = append(appArgs, []byte(arg))
	}

	tx, err := future.MakeApplicationOptInTx(
		c.AppID,
		appArgs,
		c.Accounts,
		c.ForeignApps,
		c.ForeignAssets,
		txParams,
		c.Sender.Address,
		c.Note,
		c.Group,
		c.Lease,
		c.RekeyTo,
	)
	require.NoError(t, err)

	return future.TransactionWithSigner{
		Txn: tx,
		Signer: future.BasicAccountTransactionSigner{
			Account: c.Sender,
		},
	}
}

type TxAppCall struct {
	AppID uint64
	Method string
	Args [][]byte
	Accounts []string
	ForeignApps []uint64
	ForeignAssets []uint64
	Note []byte
	Group types.Digest
	Lease [32]byte
	RekeyTo types.Address
	Sender crypto.Account // Tx sender will be the signer of this tx
}

func (c TxAppCall) Create(t *testing.T) (future.TransactionWithSigner) {

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	var appArgs [][]byte
	appArgs = append(appArgs, []byte(c.Method))
	for _, arg := range c.Args {
		appArgs = append(appArgs, []byte(arg))
	}

	tx, err := future.MakeApplicationNoOpTx(
		c.AppID,
		appArgs,
		c.Accounts,
		c.ForeignApps,
		c.ForeignAssets,
		txParams,
		c.Sender.Address,
		c.Note,
		c.Group,
		c.Lease,
		c.RekeyTo,
	)
	require.NoError(t, err)

	return future.TransactionWithSigner{
		Txn: tx,
		Signer: future.BasicAccountTransactionSigner{
			Account: c.Sender,
		},
	}
}

type TxPayment struct {
	From crypto.Account // Tx signer will be the from account
	To types.Address
	Amount uint64
	Note string
	CloseRemainderTo string
}

func (c TxPayment) Create(t *testing.T) (future.TransactionWithSigner) {

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	tx, err := future.MakePaymentTxn(
		c.From.Address.String(),
		c.To.String(),
		c.Amount,
		[]byte(c.Note),
		c.CloseRemainderTo,
		txParams,
	)
	require.NoError(t, err)

	return future.TransactionWithSigner{
		Txn: tx,
		Signer: future.BasicAccountTransactionSigner{
			Account: c.From,
		},
	}
}

func broadcastTxsAndWait(t *testing.T, txs ...TxCreator) []string {

	var txGroupBuilder future.AtomicTransactionComposer
	for _, tx := range txs {
		txGroupBuilder.AddTransaction(
			tx.Create(t),
		)
	}
	algod := algodClient(t)
	execRes, err := txGroupBuilder.Execute(algod, context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, len(txs), len(execRes.TxIDs))
	return execRes.TxIDs
}

func requireTxBroadcastError(t *testing.T, txs ...TxCreator) {

	var txGroupBuilder future.AtomicTransactionComposer
	for _, tx := range txs {
		txGroupBuilder.AddTransaction(
			tx.Create(t),
		)
	}
	algod := algodClient(t)
	_, err := txGroupBuilder.Execute(algod, context.Background(), 2)
	require.Error(t, err)
}

func getAppLocalState(t *testing.T, appID uint64, address types.Address) map[string]string {

	algodCl := algodClient(t)
	accAppInfo, err := algodCl.AccountApplicationInformation(address.String(), appID).Do(context.Background())
	require.NoError(t, err)
	return getAppStateAsMap(t, accAppInfo.AppLocalState.KeyValue)
}

func getAppGlobalState(t *testing.T, appID uint64) map[string]string {

	algodCl := algodClient(t)
	appInfo, err := algodCl.GetApplicationByID(appID).Do(context.Background())
	require.NoError(t, err)
	return getAppStateAsMap(t, appInfo.Params.GlobalState)
}

func getAppStateAsMap(t *testing.T, state []models.TealKeyValue) map[string]string {

	stateMap := make(map[string]string)
	for _, kv := range state {

		key, err := base64.StdEncoding.DecodeString(kv.Key)
		require.NoError(t, err)

		var strVal string
		if kv.Value.Type == 1 {
			strVal = string(kv.Value.Bytes)
		} else if kv.Value.Type == 2 {
			strVal = strconv.FormatUint(kv.Value.Uint, 10)
		} else {
			require.Fail(t, "unknown type")
		}

		stateMap[string(key)] = strVal
	}

	return stateMap
}

func b64StrToBytes(t *testing.T, s string) []byte {

	b, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)
	return b
}

func uint64ToBytes(t *testing.T, u uint64) []byte {

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}
