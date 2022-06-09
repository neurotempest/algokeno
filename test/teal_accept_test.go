package test

import (
	"context"
	"testing"
	"os"
	"flag"
	"fmt"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
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

     //"SOnnrX/SUSd5g0Bv8dcZW/ltR/hlaCq8BqjRBrCqeBY="
	s := "SOnnrX/SUSd5g0Bv8dcZW/ltR/hlaCq8BqjRBrCqeBY="

	fmt.Println(string(b64StrToBytes(t, s)))
}

type InnerTx struct {
	Type types.TxType
	From types.Address
	To types.Address
	Amount uint64
}

func TestContractWithSingleWinningTicket(t *testing.T) {

	creator := crypto.GenerateAccount()
	acc1 := crypto.GenerateAccount()
	acc2 := crypto.GenerateAccount()
	acc3 := crypto.GenerateAccount()

	deployedAppIDs := fundAccountsAndDeployContracts(t, 2, creator, acc1, acc2, acc3)
	require.Equal(t, 2, len(deployedAppIDs))
	appID := deployedAppIDs[0]
	appAddr := crypto.GetApplicationAddress(appID)
	nextAppID := deployedAppIDs[1]
	nextAppAddr := crypto.GetApplicationAddress(nextAppID)

	fmt.Println("deployed apps:")
	fmt.Println(appID, appAddr)
	fmt.Println(nextAppID, nextAppAddr)


	fmt.Println("Creator:")
	fmt.Println("addr:", creator.Address.String())
	fmt.Println("pub key:", base64.StdEncoding.EncodeToString(creator.PublicKey))
	fmt.Println("priv key:", base64.StdEncoding.EncodeToString(creator.PrivateKey))

	testCases := []struct{
		Name string
		Txs []TxCreator
		//Sender crypto.Account
		ExpectTxBroadcastError bool
		ExpectedLocalState map[types.Address]map[string]string
		ExpectedGlobalState map[string]string
		ExpectedInnerTxs [][]InnerTx
	}{
		{
			Name: "inital global state",
			ExpectedGlobalState: map[string]string{
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
				TxAppOptIn{
					AppID: appID,
					Sender: acc2,
				},
				TxAppOptIn{
					AppID: appID,
					Sender: acc3,
				},
			},
			ExpectedLocalState: map[types.Address]map[string]string{
				acc1.Address: {
					"wager": "0",
					"commitment": "",
				},
				acc2.Address: {
					"wager": "0",
					"commitment": "",
				},
				acc3.Address: {
					"wager": "0",
					"commitment": "",
				},
			},
			ExpectedGlobalState: map[string]string{
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
			ExpectTxBroadcastError: true,
		},
		{
			Name: "calling commit from acc1 with payment tx succeeds",
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
					To: appAddr,
					Amount: 1000000,
				},
			},
			ExpectedLocalState: map[types.Address]map[string]string{
				acc1.Address: {
					"wager": "1000000",
					"commitment": "abcdefA=",
				},
			},
			ExpectedGlobalState: map[string]string{
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
			Name: "calling commit from acc2 with payment tx succeeds",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc2,
					Method: "Commit",
					Args: [][]byte{
						b64StrToBytes(t, "ghijklA="),
					},
				},
				TxPayment{
					From: acc2,
					To: appAddr,
					Amount: 1000000,
				},
			},
			ExpectedLocalState: map[types.Address]map[string]string{
				acc1.Address: {
					"wager": "1000000",
					"commitment": "abcdefA=",
				},
			},
			ExpectedGlobalState: map[string]string{
				"numTickets": "2",
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
			ExpectTxBroadcastError: true,
		},
		{
			Name: "creator sets draw succeeds - rollover amount not send because amount too small",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: creator,
					Method: "SetDraw",
					Args: [][]byte{
						b64StrToBytes(t, "abcdefA="),
						uint64ToBytes(t, 0), // 1s
						uint64ToBytes(t, 10001),
						uint64ToBytes(t, 0), // 2s
						uint64ToBytes(t, 10002),
						uint64ToBytes(t, 0), // 3s
						uint64ToBytes(t, 10003),
						uint64ToBytes(t, 0), // 4s
						uint64ToBytes(t, 10004),
						uint64ToBytes(t, 0), // 5s
						uint64ToBytes(t, 10005),
						uint64ToBytes(t, 1), // 6s
						uint64ToBytes(t, 500000),
					},
					ForeignApps: []uint64{
						nextAppID,
					},
					Accounts: []string{
						nextAppAddr.String(),
					},
					FlatFee: types.MicroAlgos(3000),
				},
			},
			ExpectedGlobalState: map[string]string{
				"numTickets": "2",
				"draw": "abcdefA=",
				"1s": "0",
				"1p": "10001",
				"2s": "0",
				"2p": "10002",
				"3s": "0",
				"3p": "10003",
				"4s": "0",
				"4p": "10004",
				"5s": "0",
				"5p": "10005",
				"6s": "1",
				"6p": "500000",
			},
			ExpectedInnerTxs: [][]InnerTx{
				{
					{
						Type: types.PaymentTx,
						From: appAddr,
						To: creator.Address,
						Amount: 200000,
					},
				},
			},
		},
		{
			Name: "calling claim from acc2 fails because it did not win",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc2,
					Method: "Claim",
					FlatFee: types.MicroAlgos(2000),
				},
			},
			ExpectedGlobalState: map[string]string{
				"numTickets": "2",
				"draw": "abcdefA=",
				"1s": "0",
				"1p": "10001",
				"2s": "0",
				"2p": "10002",
				"3s": "0",
				"3p": "10003",
				"4s": "0",
				"4p": "10004",
				"5s": "0",
				"5p": "10005",
				"6s": "1",
				"6p": "500000",
			},
			ExpectTxBroadcastError: true,
		},
		{
			Name: "calling claim from acc1 succeeds - sends pool to acc1",
			Txs: []TxCreator{
				TxAppCall{
					AppID: appID,
					Sender: acc1,
					Method: "Claim",
					FlatFee: types.MicroAlgos(2000),
				},
			},
			ExpectedGlobalState: map[string]string{
				"numTickets": "2",
				"draw": "abcdefA=",
				"1s": "0",
				"1p": "10001",
				"2s": "0",
				"2p": "10002",
				"3s": "0",
				"3p": "10003",
				"4s": "0",
				"4p": "10004",
				"5s": "0",
				"5p": "10005",
				"6s": "1",
				"6p": "500000",
			},
			ExpectedInnerTxs: [][]InnerTx{
				{
					{
						Type: types.PaymentTx,
						From: appAddr,
						To: acc1.Address,
						Amount: 500000,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			if test.ExpectTxBroadcastError {
				requireTxBroadcastError(t, test.Txs...)
				return
			}

			var txIDs []string
			if len(test.Txs) > 0 {
				txIDs = broadcastTxsAndWait(t, test.Txs...)
			}

			for addr, expectedLocalState := range test.ExpectedLocalState {
				localState := getAppLocalState(t, appID, addr)
				require.Equal(t, expectedLocalState, localState)
			}

			if len(test.ExpectedGlobalState) > 0 {
				globalState := getAppGlobalState(t, appID)
				require.Equal(t, test.ExpectedGlobalState, globalState)
			}

			if len(test.ExpectedInnerTxs) > 0 {
				require.Equal(t, len(test.ExpectedInnerTxs), len(txIDs))

				for _, expectedInnterTxs := range test.ExpectedInnerTxs {
					if len(expectedInnterTxs) > 0 {
						algodCl := algodClient(t)
						pendingRes, _, err := algodCl.PendingTransactionInformation(txIDs[len(txIDs)-1]).Do(context.Background())
						require.NoError(t, err)

						require.Equal(t, len(expectedInnterTxs), len(pendingRes.InnerTxns))
						for iTx, expected := range expectedInnterTxs {
							actual := pendingRes.InnerTxns[iTx].Transaction.Txn
							require.Equal(t, expected.Type, actual.Type)
							require.Equal(t, expected.From, actual.Sender)
							require.Equal(t, expected.To, actual.Receiver)
							require.Equal(t, expected.Amount, uint64(actual.Amount), "Amounts not equal: %i != %i", int64(expected.Amount), int64(actual.Amount))
						}
					}
				}
			}
		})
	}
}

// TODO: Add test where there are many account have tickets so that the rollover amount is greater than the minimum account amount
// and the rollover amount is send to the next contract, and all of the tickets claim their prizes

func fundAccountsAndDeployContracts(
	t *testing.T,
	numContracts int,
	creator crypto.Account,
	accounts ...crypto.Account,
) []uint64 {

	kmdAcc := getKMDAccount(t)

	fundAmount := uint64(20_000_000)

	var txs []TxCreator
	txs = append(txs, TxPayment{
		From: kmdAcc,
		To: creator.Address,
		Amount: fundAmount,
	})

	for _, acc := range accounts {
		txs = append(txs, TxPayment{
			From: kmdAcc,
			To: acc.Address,
			Amount: fundAmount,
		})
	}

	for i:=0; i<numContracts; i++ {
		txs = append(txs, TxAppDeploy{
			Creator: creator,
			ApprovalPath: "../contract/approval.teal",
			ClearPath: "../contract/clear.teal",
			SchemaPath: "../contract/schema.json",
			Note: uint64ToBytes(t, uint64(i)),
		})
	}

	txIDs := broadcastTxsAndWait(t, txs...)

	var deployedAppIDs []uint64
	algodCl := algodClient(t)
	for iRes:=0; iRes<numContracts; iRes++ {
		pendingRes, _, err := algodCl.PendingTransactionInformation(
			txIDs[len(txIDs)-(numContracts - iRes)],
		).Do(context.Background())
		require.NoError(t, err)

		deployedAppIDs = append(deployedAppIDs, pendingRes.ApplicationIndex)
	}

	return deployedAppIDs
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

type TealSchema struct {
	GlobalByteSlices uint64 `json:"global_byte_slices"`
	GlobalUints uint64 `json:"global_uints"`
	LocalByteSlices uint64 `json:"local_byte_slices"`
	LocalUints uint64 `json:"local_uints"`
}

func readSchemaFile(t *testing.T, schemaPath string) TealSchema {
	b, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	var s TealSchema
	json.Unmarshal(b, &s)
	return s
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
	SchemaPath string
	Note []byte
}

func (c TxAppDeploy) Create(t *testing.T) (future.TransactionWithSigner) {

	s := readSchemaFile(t, c.SchemaPath)
	appCreate := TxAppCreate{
		ApprovalProg: compileTeal(t, c.ApprovalPath),
		ClearProg: compileTeal(t, c.ClearPath),
		GlobalUints: s.GlobalUints,
		GlobalByteSlices: s.GlobalByteSlices,
		LocalUints: s.LocalUints,
		LocalByteSlices: s.LocalByteSlices,
		Note: c.Note,
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

	FlatFee types.MicroAlgos

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

	if c.FlatFee != 0 {
		txParams.Fee = c.FlatFee
		txParams.FlatFee = true
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
