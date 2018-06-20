package app

import (
	"os"
	"encoding/json"
	"errors"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	crypto "github.com/tendermint/go-crypto"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/wire"

	"github.com/FourthState/plasma-mvp-sidechain/types"
	"github.com/ethereum/go-ethereum/common"
)

// State to Unmarshal
type GenesisState struct {
	UTXOs  []types.UTXO   `json:"UTXOs"`
}

func NewGenesisUTXO(addr common.Address, amount uint64, position [4]uint64) types.UTXO {
	utxo := types.BaseUTXO{
		Address: addr,
		Denom: amount,
	}
	utxo.SetPosition(position[0], uint16(position[1]), uint8(position[2]), position[3])
	return utxo
}

var (
	flagAddress = "address"
	flagClientHome = "home-client"
	flagOWK = "owk"

	// UTXO amount awarded
	freeEtherVal    = int64(100)

	// default home directories for expected binaries
	DefaultCLIHome  = os.ExpandEnv("$HOME/.plasmacli")
	DefaultNodeHome = os.ExpandEnv("$HOME/.plasmad")
)

// get app init parameters for server init command
func PlasmaAppInit() server.AppInit {
	fsAppGenState := pflag.NewFlagSet("", pflag.ContinueOnError)

	fsAppGenTx := pflag.NewFlagSet("", pflag.ContinueOnError)
	fsAppGenTx.String(flagAddress, "", "address, required")
	fsAppGenTx.String(flagClientHome, DefaultCLIHome,
		"home directory for the client, used for key generation")
	fsAppGenTx.Bool(flagOWK, false, "overwrite the accounts created")

	return server.AppInit{
		FlagsAppGenState: fsAppGenState,
		FlagsAppGenTx:    fsAppGenTx,
		AppGenTx:         PlasmaAppGenTx,
		AppGenState:      GaiaAppGenStateJSON,
	}
}

// simple genesis tx
type PlasmaGenTx struct {
	Address common.Address   `json:"address"`
}

// Generate a gaia genesis transaction with flags
func PlasmaAppGenTx(cdc *wire.Codec, pk crypto.PubKey) (
	appGenTx, cliPrint json.RawMessage, validator tmtypes.GenesisValidator, err error) {
	addrString := viper.GetString(flagAddress)
	overwrite := viper.GetBool(flagOWK)

	addr := common.HexToAddress(addrString)
	if err != nil {
		return
	}
	cliPrint = json.RawMessage("success")
	appGenTx,_,validator,err = PlasmaAppGenTxNF(cdc, pk, addr, overwrite)
	return
}

// Generate a gaia genesis transaction without flags
func PlasmaAppGenTxNF(cdc *wire.Codec, pk crypto.PubKey, addr common.Address, overwrite bool) (
	appGenTx, cliPrint json.RawMessage, validator tmtypes.GenesisValidator, err error) {

	var bz []byte
	plasmaGenTx := PlasmaGenTx{
		Address: addr,
	}
	bz, err = wire.MarshalJSONIndent(cdc, plasmaGenTx)
	if err != nil {
		return
	}
	appGenTx = json.RawMessage(bz)

	validator = tmtypes.GenesisValidator{
		PubKey: pk,
		Power:  1,
	}
	return
}

// Create the core parameters for genesis initialization for gaia
// note that the pubkey input is this machines pubkey
func PlasmaAppGenState(cdc *wire.Codec, appGenTxs []json.RawMessage) (genesisState GenesisState, err error) {

	if len(appGenTxs) == 0 {
		err = errors.New("must provide at least genesis transaction")
		return
	}

	// get genesis flag account information
	genUTXO := make([]types.UTXO, len(appGenTxs))
	for i, appGenTx := range appGenTxs {

		var genTx PlasmaGenTx
		err = cdc.UnmarshalJSON(appGenTx, &genTx)
		if err != nil {
			return
		}

		genUTXO[i] = NewGenesisUTXO(genTx.Address, 100, [4]uint64{0, 0, 0, uint64(i + 1)})
	}

	// create the final app state
	genesisState = GenesisState{
		UTXOs: genUTXO,
	}
	return
}

// GaiaAppGenState but with JSON
func GaiaAppGenStateJSON(cdc *wire.Codec, appGenTxs []json.RawMessage) (appState json.RawMessage, err error) {

	// create the final app state
	genesisState, err := PlasmaAppGenState(cdc, appGenTxs)
	if err != nil {
		return nil, err
	}
	appState, err = wire.MarshalJSONIndent(cdc, genesisState)
	return
}