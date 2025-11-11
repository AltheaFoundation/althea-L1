package canto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gogo/protobuf/proto"

	"github.com/AltheaFoundation/althea-L1/x/erc20/types/canto"
)

func TestRegisterCoinProposal(t *testing.T) {
	metadata := banktypes.Metadata{
		Description: "Test token",
		Base:        "utest",
		Display:     "test",
		Name:        "Test Token",
		Symbol:      "TEST",
		URI:         "",
		URIHash:     "",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "utest",
				Exponent: 0,
			},
			{
				Denom:    "test",
				Exponent: 6,
			},
		},
	}

	proposal := &canto.RegisterCoinProposal{
		Title:       "Register Test Token",
		Description: "This is a test proposal",
		Metadata:    metadata,
	}

	// Test ProposalRoute
	require.Equal(t, "erc20", proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, "RegisterCoin", proposal.ProposalType())

	// Test ValidateBasic with valid metadata
	err := proposal.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic with invalid metadata (empty base)
	invalidProposal := &canto.RegisterCoinProposal{
		Title:       "Invalid Proposal",
		Description: "This proposal has invalid metadata",
		Metadata: banktypes.Metadata{
			Base:        "",
			Description: "",
			DenomUnits:  nil,
			Display:     "",
			Name:        "",
			Symbol:      "",
			URI:         "",
			URIHash:     "",
		},
	}
	err = invalidProposal.ValidateBasic()
	require.Error(t, err)

	// Test marshaling/unmarshaling
	bz, err := proto.Marshal(proposal)
	require.NoError(t, err)

	unmarshaled := &canto.RegisterCoinProposal{
		Title:       "",
		Description: "",
		Metadata: banktypes.Metadata{
			Description: "",
			Base:        "",
			Display:     "",
			Name:        "",
			Symbol:      "",
			URI:         "",
			URIHash:     "",
			DenomUnits:  nil,
		},
	}
	err = proto.Unmarshal(bz, unmarshaled)
	require.NoError(t, err)
	require.Equal(t, proposal.Title, unmarshaled.Title)
	require.Equal(t, proposal.Description, unmarshaled.Description)
	require.Equal(t, proposal.Metadata.Base, unmarshaled.Metadata.Base)
}

func TestRegisterERC20Proposal(t *testing.T) {
	validAddress := "0x1234567890123456789012345678901234567890"

	proposal := &canto.RegisterERC20Proposal{
		Title:        "Register ERC20",
		Description:  "Register an ERC20 token",
		Erc20Address: validAddress,
	}

	// Test ProposalRoute
	require.Equal(t, "erc20", proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, "RegisterERC20", proposal.ProposalType())

	// Test ValidateBasic with valid address
	err := proposal.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic with invalid address
	invalidProposal := &canto.RegisterERC20Proposal{
		Title:        "Invalid Proposal",
		Description:  "This has an invalid address",
		Erc20Address: "invalid",
	}
	err = invalidProposal.ValidateBasic()
	require.Error(t, err)

	// Test marshaling/unmarshaling
	bz, err := proto.Marshal(proposal)
	require.NoError(t, err)

	unmarshaled := &canto.RegisterERC20Proposal{
		Title:        "",
		Description:  "",
		Erc20Address: "",
	}
	err = proto.Unmarshal(bz, unmarshaled)
	require.NoError(t, err)
	require.Equal(t, proposal.Title, unmarshaled.Title)
	require.Equal(t, proposal.Description, unmarshaled.Description)
	require.Equal(t, proposal.Erc20Address, unmarshaled.Erc20Address)
}

func TestToggleTokenConversionProposal(t *testing.T) {
	validAddress := "0x1234567890123456789012345678901234567890"

	proposal := &canto.ToggleTokenConversionProposal{
		Title:       "Toggle Conversion",
		Description: "Toggle token conversion",
		Token:       validAddress,
	}

	// Test ProposalRoute
	require.Equal(t, "erc20", proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, "ToggleTokenConversion", proposal.ProposalType())

	// Test ValidateBasic with valid address
	err := proposal.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic with valid denom
	denomProposal := &canto.ToggleTokenConversionProposal{
		Title:       "Toggle Conversion",
		Description: "Toggle token conversion",
		Token:       "uatom",
	}
	err = denomProposal.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic with invalid token
	invalidProposal := &canto.ToggleTokenConversionProposal{
		Title:       "Invalid Proposal",
		Description: "This has an invalid token",
		Token:       "!!!invalid!!!",
	}
	err = invalidProposal.ValidateBasic()
	require.Error(t, err)

	// Test marshaling/unmarshaling
	bz, err := proto.Marshal(proposal)
	require.NoError(t, err)

	unmarshaled := &canto.ToggleTokenConversionProposal{
		Title:       "",
		Description: "",
		Token:       "",
	}
	err = proto.Unmarshal(bz, unmarshaled)
	require.NoError(t, err)
	require.Equal(t, proposal.Title, unmarshaled.Title)
	require.Equal(t, proposal.Description, unmarshaled.Description)
	require.Equal(t, proposal.Token, unmarshaled.Token)
}

// TestBackwardCompatibility ensures that the canto types can deserialize
// proposals with the canto.erc20.v1 type URL, simulating historical proposals.
func TestBackwardCompatibility(t *testing.T) {
	// This test verifies that proposals marshaled with the canto package
	// can be successfully unmarshaled, ensuring backward compatibility
	// with historical on-chain proposals.

	metadata := banktypes.Metadata{
		Description: "Historical token",
		Base:        "uhistorical",
		Display:     "historical",
		Name:        "Historical Token",
		Symbol:      "HIST",
		URI:         "",
		URIHash:     "",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "uhistorical",
				Exponent: 0,
			},
			{
				Denom:    "historical",
				Exponent: 6,
			},
		},
	}

	// Create a proposal as it would have been stored historically
	historicalProposal := &canto.RegisterCoinProposal{
		Title:       "Historical Proposal",
		Description: "A proposal from before the Cardinal upgrade",
		Metadata:    metadata,
	}

	// Marshal it (simulating how it was stored on-chain)
	bz, err := proto.Marshal(historicalProposal)
	require.NoError(t, err)

	// Unmarshal it (simulating a query of historical data)
	retrieved := &canto.RegisterCoinProposal{
		Title:       "",
		Description: "",
		Metadata: banktypes.Metadata{
			Description: "",
			Base:        "",
			Display:     "",
			Name:        "",
			Symbol:      "",
			URI:         "",
			URIHash:     "",
			DenomUnits:  nil,
		},
	}
	err = proto.Unmarshal(bz, retrieved)
	require.NoError(t, err)

	// Verify the data is intact
	require.Equal(t, historicalProposal.Title, retrieved.Title)
	require.Equal(t, historicalProposal.Description, retrieved.Description)
	require.Equal(t, historicalProposal.Metadata.Base, retrieved.Metadata.Base)

	// Verify validation still works
	err = retrieved.ValidateBasic()
	require.NoError(t, err)
}
