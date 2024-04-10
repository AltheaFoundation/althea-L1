package cli

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/Canto-Network/Canto/v5/x/govshuttle/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// PARSING METADATA ACCORDING TO PROPOSAL STRUCT IN GOVTYPES TYPE IN NATIVEDEX

func ParseTreasuryMetadata(cdc codec.JSONCodec, metadataFile string) (types.TreasuryProposalMetadata, error) {
	propMetaData := types.TreasuryProposalMetadata{}

	contents, err := ioutil.ReadFile(filepath.Clean(metadataFile))
	if err != nil {
		return propMetaData, err
	}

	// if err = cdc.UnmarshalJSON(contents, &propMetaData); err != nil {
	// 	return propMetaData, err
	// }

	if err = json.Unmarshal(contents, &propMetaData); err != nil {
		return types.TreasuryProposalMetadata{}, err
	}

	propMetaData.PropID = 0

	return propMetaData, nil
}
