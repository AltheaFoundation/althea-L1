package althea

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	cryptocodec "github.com/tendermint/tendermint/crypto/encoding"
	tmtypes "github.com/tendermint/tendermint/types"
)

type replacementConfigs []replacementConfig

func (r *replacementConfigs) isReplacedValidator(validatorAddress string) (int, replacementConfig) {

	for i, replacement := range *r {
		if replacement.ValidatorAddress == validatorAddress {
			return i, replacement
		}
	}

	// nolint: exhaustruct
	return -1, replacementConfig{}
}

type replacementConfig struct {
	Name             string `json:"validator_name"`
	ValidatorAddress string `json:"validator_address"`
	ConsensusPubkey  string `json:"stargate_consensus_public_key"`
}

func loadKeydataFromFile(clientCtx client.Context, replacementrJSON string, genDoc *tmtypes.GenesisDoc) *tmtypes.GenesisDoc {
	jsonReplacementBlob, err := os.ReadFile(replacementrJSON)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "failed to read replacement keys from file %s", replacementrJSON))
	}

	var replacementKeys replacementConfigs

	err = json.Unmarshal(jsonReplacementBlob, &replacementKeys)

	if err != nil {
		log.Fatal("Could not unmarshal replacement keys ")
	}

	var state types.AppMap
	if err := json.Unmarshal(genDoc.AppState, &state); err != nil {
		log.Fatal(errors.Wrap(err, "failed to JSON unmarshal initial genesis state"))
	}

	var stakingGenesis staking.GenesisState
	var slashingGenesis slashing.GenesisState

	clientCtx.Codec.MustUnmarshalJSON(state[staking.ModuleName], &stakingGenesis)
	clientCtx.Codec.MustUnmarshalJSON(state[slashing.ModuleName], &slashingGenesis)

	for i, val := range stakingGenesis.Validators {
		idx, replacement := replacementKeys.isReplacedValidator(val.OperatorAddress)

		if idx != -1 {

			toReplaceValConsAddress, err := val.GetConsAddr()
			if err != nil {
				panic(sdkerrors.Wrapf(err, "unable to get validator cons addr for validator %v", val.Description))
			}

			consPubKeyBz, err := sdk.GetFromBech32(replacement.ConsensusPubkey, sdk.GetConfig().GetBech32ConsensusPubPrefix())
			if err != nil {
				log.Fatal(fmt.Errorf("failed to decode key:%s %w", consPubKeyBz, err))
			}

			consPubKey, err := legacy.PubKeyFromBytes(consPubKeyBz)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to decode key:%s %w", consPubKey, err))
			}

			val.ConsensusPubkey, err = codectypes.NewAnyWithValue(consPubKey)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to decode key:%s %w", consPubKey, err))
			}

			replaceValConsAddress, err := val.GetConsAddr()
			if err != nil {
				panic(sdkerrors.Wrapf(err, "unable to get new validator cons addr for pubkey %v", consPubKey.String()))
			}

			// nolint: errcheck
			protoReplaceValConsPubKey, err := val.TmConsPublicKey()
			if err != nil {
				panic(sdkerrors.Wrapf(err, "unable to get validator tm cons public key for  validator %v", val.Description))
			}
			// nolint: errcheck
			replaceValConsPubKey, err := cryptocodec.PubKeyFromProto(protoReplaceValConsPubKey)
			if err != nil {
				panic(sdkerrors.Wrapf(err, "unable to decode pubkey %v", protoReplaceValConsPubKey.String()))
			}

			for i, signingInfo := range slashingGenesis.SigningInfos {
				if signingInfo.Address == toReplaceValConsAddress.String() {
					slashingGenesis.SigningInfos[i].Address = replaceValConsAddress.String()
					slashingGenesis.SigningInfos[i].ValidatorSigningInfo.Address = replaceValConsAddress.String()
				}
			}

			for i, missedInfo := range slashingGenesis.MissedBlocks {
				if missedInfo.Address == toReplaceValConsAddress.String() {
					slashingGenesis.MissedBlocks[i].Address = replaceValConsAddress.String()
				}
			}

			for tmIdx, tmval := range genDoc.Validators {
				if tmval.Address.String() == replaceValConsAddress.String() {
					genDoc.Validators[tmIdx].Address = replaceValConsAddress.Bytes()
					genDoc.Validators[tmIdx].PubKey = replaceValConsPubKey

				}
			}
			stakingGenesis.Validators[i] = val

		}

	}
	state[staking.ModuleName] = clientCtx.Codec.MustMarshalJSON(&stakingGenesis)
	state[slashing.ModuleName] = clientCtx.Codec.MustMarshalJSON(&slashingGenesis)

	genDoc.AppState, err = json.Marshal(state)

	if err != nil {
		log.Fatal("Could not marshal App State")
	}
	return genDoc

}
