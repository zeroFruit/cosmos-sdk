package cmd

import (
	"fmt"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/tendermint/go-amino"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/cosmos-sdk/server/rosetta"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	crg "github.com/tendermint/cosmos-rosetta-gateway/server"
)

const (
	flagBlockchain    = "blockchain"
	flagNetwork       = "network"
	flagTendermintRPC = "tendermint-rpc"
	flagListenAddr    = "listen-addr"
)

// RosettaCommand will start the application Rosetta API service as a blocking process.
func RosettaCommand(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use: "rosetta",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := getRosettaSettingsFromFlags(cdc, cmd.Flags())
			if err != nil {
				return err
			}
			srv, err := crg.NewServer(settings)
			if err != nil {
				return err
			}
			return srv.Start()
		},
	}

	cmd.Flags().String(flagBlockchain, "blockchain", "Application's name (e.g. Cosmos Hub)")
	cmd.Flags().String(flagListenAddr, "localhost:8080", "The address where Rosetta API will listen.")
	cmd.Flags().String(flagNetwork, "network", "Network's identifier (e.g. cosmos-hub-3, testnet-1, etc)")
	cmd.Flags().String(flagTendermintRPC, "localhost:26657", "Tendermint's RPC endpoint.")

	return cmd
}

func getRosettaSettingsFromFlags(cdc *amino.Codec, flags *flag.FlagSet) (crg.Settings, error) {
	listenAddr, err := flags.GetString(flagListenAddr)
	if err != nil {
		return crg.Settings{}, err
	}
	blockchain, err := flags.GetString(flagBlockchain)
	if err != nil {
		return crg.Settings{}, fmt.Errorf("invalid blockchain value: %w", err)
	}

	network, err := flags.GetString(flagNetwork)
	if err != nil {
		return crg.Settings{}, fmt.Errorf("invalid network value: %w", err)
	}

	tendermintRPC, err := flags.GetString(flagTendermintRPC)
	if err != nil {
		return crg.Settings{}, fmt.Errorf("invalid tendermint rpc value: %w", err)
	}

	if !strings.HasPrefix(tendermintRPC, "tcp://") {
		tendermintRPC = fmt.Sprintf("tcp://%s", tendermintRPC)
	}
	client, err := rosetta.NewClient(tendermintRPC, cdc)
	if err != nil {
		return crg.Settings{}, err
	}
	return crg.Settings{
		Network: &types.NetworkIdentifier{
			Blockchain: blockchain,
			Network:    network,
		},
		Client:    client,
		Listen:    listenAddr,
		Offline:   false,
		Retries:   5,
		RetryWait: 5 * time.Second,
	}, nil
}
