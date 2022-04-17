package test

import (
	"context"
	"testing"
	"os"
	"flag"
	"fmt"
	"encoding/base64"
	"encoding/binary"
	"log"
	"strconv"

	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/stretchr/testify/require"

	"encoding/ascii85"
)

var (
	algodHost = flag.String("algod_host", "http://localhost:4001", "Host of algod client")
	algodTokenPath = flag.String("algod_token_path", "../algorand/algod.token", "Path to algod token")
	kmdHost = flag.String("kmd_host", "http://localhost:4002", "Host of kmd client")
	kmdTokenPath = flag.String("kmd_token_path", "../algorand/kmd.token", "Path to kmd token")
	indexerHost = flag.String("indexer_host", "http://localhost:4003", "Host of indexer client")
)

func TestContract(t *testing.T) {

	creator := crypto.GenerateAccount()
	acc1 := crypto.GenerateAccount()
	acc2 := crypto.GenerateAccount()
	acc3 := crypto.GenerateAccount()

	appID := fundAccountsAndDeployContract(t, creator, acc1, acc2, acc3)

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
		Amount: 1000000,
	})

	for _, acc := range accounts {
		txs = append(txs, TxPayment{
			From: kmdAcc,
			To: acc.Address,
			Amount: 1000000,
		})
	}

	txs = append(txs, TxAppDeploy{
		Creator: creator,
		ApprovalPath: "../contract/approval.teal",
		ClearPath: "../contract/clear.teal",
		GlobalUints: 1,
		GlobalByteSlices: 1,
		LocalUints: 1,
		LocalByteSlices:1,
	})

	txIDs := broadcastTxsAndWait(t, txs...)

	algodCl := algodClient(t)
	pendingRes, _, err := algodCl.PendingTransactionInformation(txIDs[len(txIDs)-1]).Do(context.Background())
	require.NoError(t, err)

	return pendingRes.ApplicationIndex
}


func TestDeployContract(t *testing.T) {

	ownerAccount := crypto.GenerateAccount()
	otherAccount := crypto.GenerateAccount()

	fmt.Println("Other acc:")
	fmt.Println(otherAccount.Address)
	fmt.Println(base64.StdEncoding.EncodeToString(otherAccount.PublicKey))
	fmt.Println(base64.StdEncoding.EncodeToString(otherAccount.PrivateKey))

	fmt.Println(ownerAccount.Address.String())

	log.Println("funding...")
	appID := fundAccountsAndDeployContract(t, ownerAccount, otherAccount)


	log.Println(otherAccount.Address, "opting_in to", appID, "...")
	broadcastTxsAndWait(
		t,
		TxAppOptIn{
			AppID: appID,
			Sender: otherAccount,
		},
	)

	amt := uint64(120000)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, amt)

	log.Println(otherAccount.Address, "calling Commit...")
	broadcastTxsAndWait(
		t,
		TxAppCall{
			AppID: appID,
			Method: "Commit",
			Args: [][]byte{
				[]byte("abcdef"),
			},
			Sender: otherAccount,
		},
		TxPayment{
			From: otherAccount,
			To: crypto.GetApplicationAddress(appID),
			Amount: amt,
		},
	)

}

func TestPokeExisting(t *testing.T) {

	addr, err := types.DecodeAddress("KSHZJX64NLFOIQFJIDYULNR37THQ3EHEVVPR2EJ2C2VB7YBSN625PJM5GE")
	require.NoError(t, err)
	pubk, err := base64.StdEncoding.DecodeString("VI+U39xqyuRAqUDxRbY7/M8NkOStXx0ROhaqH+Ayb7U=")
	require.NoError(t, err)
	privk, err := base64.StdEncoding.DecodeString("u+Eb7y2VolnhQvEYE/fNdKmieRhY9Y/ckSeLX9NVgldUj5Tf3GrK5ECpQPFFtjv8zw2Q5K1fHRE6Fqof4DJvtQ==")
	require.NoError(t, err)

	acc := crypto.Account{
		Address: addr,
		PublicKey: pubk,
		PrivateKey: privk,
	}

	appID := uint64(63)

	fmt.Println("app addr:", crypto.GetApplicationAddress(appID))

	broadcastTxsAndWait(
		t,
		TxAppCall{
			AppID: appID,
			Method: "Commit",
			Args: [][]byte{
				[]byte("ZZoo=="),
			},
			Sender: acc,
		},
		TxPayment{
			From: acc,
			To: crypto.GetApplicationAddress(appID),
			Amount: 100000,
		},
	)

	algod := algodClient(t)

	accAppInfo, err := algod.AccountApplicationInformation(acc.Address.String(), appID).Do(context.Background())
	require.NoError(t, err)

	appInfo, err := algod.GetApplicationByID(appID).Do(context.Background())

	fmt.Println("accAppInfo:")
	fmt.Println(" - AppId:", accAppInfo.AppLocalState.Id)
	fmt.Println(" - locals (key,val):", accAppInfo.AppLocalState.KeyValue)

	for _, loc := range accAppInfo.AppLocalState.KeyValue {

		key, err := base64.StdEncoding.DecodeString(loc.Key)
		require.NoError(t, err)

		var strVal string
		if loc.Value.Type == 1 {
			tmp, err := base64.StdEncoding.DecodeString(loc.Value.Bytes)
			require.NoError(t, err)
			strVal = string(tmp)
		} else if loc.Value.Type == 2 {
			strVal = strconv.FormatUint(loc.Value.Uint, 10)
		} else {
			require.Fail(t, "Unknown type")
		}

		fmt.Println(string(key), strVal)

	}

	fmt.Println("appGlobalState:", appInfo.Params.GlobalState)

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
