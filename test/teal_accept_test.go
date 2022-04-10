package test

import (
	"context"
	"testing"
	"os"
	"flag"
	"fmt"
	"encoding/base64"
	"crypto/ed25519"
	"log"

	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/indexer"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	algodHost = flag.String("algod_host", "http://localhost:4001", "Host of algod client")
	algodTokenPath = flag.String("algod_token_path", "../algorand/algod.token", "Path to algod token")
	kmdHost = flag.String("kmd_host", "http://localhost:4002", "Host of kmd client")
	kmdTokenPath = flag.String("kmd_token_path", "../algorand/kmd.token", "Path to kmd token")
	indexerHost = flag.String("indexer_host", "http://localhost:4003", "Host of indexer client")
)


func TestDeployContract(t *testing.T) {

	ownerAccount := crypto.GenerateAccount()
	otherAccount := crypto.GenerateAccount()

	fmt.Println(ownerAccount.Address.String())

	log.Println("funding...")
	fundMultipleAccountsFromKMDAccount(t, 1000000, ownerAccount, otherAccount)

	log.Println("deploying...")
	appID := deployContract(
		t,
		ownerAccount,
		"../contract/approval.teal",
		"../contract/clear.teal",
		0,
		1,
		1,
		1,
	)

	log.Println(otherAccount.Address, "opting_in...")
	appOptIn(t, appID, otherAccount)
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

func fundMultipleAccountsFromKMDAccount(t *testing.T, amount uint64, accounts ...crypto.Account) {

	kmdAcc := getKMDAccount(t)

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	var txGroupBuilder future.AtomicTransactionComposer

	for _, acc := range accounts {
		tx, err := future.MakePaymentTxn(
			kmdAcc.Address.String(),
			acc.Address.String(),
			amount,
			[]byte("inital funding"),
			"",
			txParams,
		)
		require.NoError(t, err)

		txGroupBuilder.AddTransaction(
			future.TransactionWithSigner{
				Txn: tx,
				Signer: future.BasicAccountTransactionSigner{
					Account: kmdAcc,
				},
			},
		)
	}

	_, err = txGroupBuilder.Execute(algod, context.Background(), 2)
	require.NoError(t, err)
}

func deployContract(
	t *testing.T,
	creatorAcc crypto.Account,
	approvalPath string,
	clearPath string,
	// TODO: Parse the teal progs to figure out how many local/global vars are needed... not sure why algod can't figure this out automatically.
	numGlobalUints uint64,
	numGlobalByteSlices uint64,
	numLocalUints uint64,
	numLocalByteSlices uint64,
) uint64 {

	approvalProg := compileTeal(t,approvalPath)
	clearProg := compileTeal(t,clearPath)

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	tx, err := future.MakeApplicationCreateTx(
		false,
		approvalProg,
		clearProg,
		types.StateSchema{
			NumUint: numGlobalUints,
			NumByteSlice: numGlobalByteSlices,
		},
		types.StateSchema{
			NumUint: numLocalUints,
			NumByteSlice: numLocalByteSlices,
		},
		nil,
		nil,
		nil,
		nil,
		txParams,
		creatorAcc.Address,
		nil,
		types.Digest{},
		[32]byte{},
		types.Address{},
	)
	require.NoError(t, err)

	var txGroupBuilder future.AtomicTransactionComposer
	txGroupBuilder.AddTransaction(
		future.TransactionWithSigner{
			Txn: tx,
			Signer: future.BasicAccountTransactionSigner{
				Account: creatorAcc,
			},
		},
	)
	execRes, err := txGroupBuilder.Execute(algod, context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, 1, len(execRes.TxIDs))

	pendignRes, _, err := algod.PendingTransactionInformation(execRes.TxIDs[0]).Do(context.Background())
	require.NoError(t, err)

	return pendignRes.ApplicationIndex
}

func appOptIn(t *testing.T, appID uint64, acc crypto.Account) {

	algod := algodClient(t)
	txParams, err := algod.SuggestedParams().Do(context.Background())
	require.NoError(t, err)

	tx, err := future.MakeApplicationOptInTx(
		appID,
		nil,
		nil,
		nil,
		nil,
		txParams,
		acc.Address,
		nil,
		types.Digest{},
		[32]byte{},
		types.Address{},
	)

	var txGroupBuilder future.AtomicTransactionComposer
	txGroupBuilder.AddTransaction(
		future.TransactionWithSigner{
			Txn: tx,
			Signer: future.BasicAccountTransactionSigner{
				Account: acc,
			},
		},
	)
	_, err = txGroupBuilder.Execute(algod, context.Background(), 2)
	require.NoError(t, err)
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

func signTxSendAndWait(t *testing.T, privKey ed25519.PrivateKey, tx types.Transaction) string {

	_, signedTx, err := crypto.SignTransaction(privKey, tx)
	require.NoError(t, err)

	algod := algodClient(t)
	txID, err := algod.SendRawTransaction(signedTx).Do(context.Background())
	require.NoError(t, err)

	_, err = future.WaitForConfirmation(algod, txID, 4, context.Background())
	require.NoError(t, err)

	return txID
}
