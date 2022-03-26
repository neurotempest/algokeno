package test

import (
	"testing"
	"os"
	"flag"
	"fmt"

	//"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/stretchr/testify/require"
)

var (
	algodHost = flag.String("algod_host", "http://localhost:4001", "Host of algod client")
	algodTokenPath = flag.String("algod_token_path", "./algorand/algod.token", "Path to algod token")
	kmdHost = flag.String("kmd_host", "http://localhost:4002", "Host of kmd client")
	kmdTokenPath = flag.String("kmd_token_path", "./algorand/kmd.tok", "Path to kmd token")
)

func TestGetPrivateKeyFromKMD(t *testing.T) {

	token, err := os.ReadFile(*kmdTokenPath)
	require.NoError(t, err)

	kmd, err := kmd.MakeClient(*kmdHost, string(token))
	require.NoError(t, err)

	w, err := kmd.ListWallets()
	require.NoError(t, err)
	fmt.Printf("%+v\n", w)

	handle, err := kmd.InitWalletHandle(w.Wallets[0].ID, "")
	require.NoError(t, err)

	k, err := kmd.ListKeys(handle.WalletHandleToken)
	require.NoError(t, err)
	fmt.Printf("%+v\n", k)

	pk, err := kmd.ExportKey(handle.WalletHandleToken, "", k.Addresses[0])
	require.NoError(t, err)
	fmt.Printf("%+v\n", pk)
}
