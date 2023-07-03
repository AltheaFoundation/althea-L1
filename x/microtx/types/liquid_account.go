package types

import (
	"math/big"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

// LiquidAccountThreshold stores the threshold limit for one token, used to control Liquid Infrastructure Accounts
type LiquidAccountThreshold struct {
	Amount big.Int
	Token  common.Address
}

func NewLiquidAccountThreshold(token common.Address, amount big.Int) LiquidAccountThreshold {
	return LiquidAccountThreshold{Token: token, Amount: amount}
}

func NewLiquidAccountThresholds(tokens []common.Address, amounts []big.Int) ([]LiquidAccountThreshold, error) {
	if len(tokens) != len(amounts) {
		return nil, sdkerrors.Wrap(ErrInvalidThresholds, "token addresses must match limiting amounts")
	}
	out := []LiquidAccountThreshold{}
	for i := 0; i < len(tokens); i++ {
		out = append(out, LiquidAccountThreshold{Token: tokens[i], Amount: amounts[i]})
	}

	return out, nil
}

// FindThresholdForERC20 identifies which of `thresholds` corresponds to `token`
// Returns nil if none are found
func FindThresholdForERC20(thresholds []LiquidAccountThreshold, token common.Address) *LiquidAccountThreshold {
	if len(thresholds) == 0 {
		return nil
	}
	for _, thold := range thresholds {
		if thold.Token == token {
			return &thold
		}
	}

	return nil
}
