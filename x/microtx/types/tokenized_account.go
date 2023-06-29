package types

import (
	"math/big"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

// TokenizedAccountThreshold stores the threshold limit for one token, used to control TokenizedAccounts
type TokenizedAccountThreshold struct {
	Amount big.Int
	Token  common.Address
}

func NewTokenizedAccountThreshold(token common.Address, amount big.Int) TokenizedAccountThreshold {
	return TokenizedAccountThreshold{Token: token, Amount: amount}
}

func NewTokenizedAccountThresholds(tokens []common.Address, amounts []big.Int) ([]TokenizedAccountThreshold, error) {
	if len(tokens) != len(amounts) {
		return nil, sdkerrors.Wrap(ErrInvalidThresholds, "token addresses must match limiting amounts")
	}
	out := []TokenizedAccountThreshold{}
	for i := 0; i < len(tokens); i++ {
		out = append(out, TokenizedAccountThreshold{Token: tokens[i], Amount: amounts[i]})
	}

	return out, nil
}

// FindThresholdIntersection determines which configured threshold ERC20s exist in `changedErc20s` for the purpose of
// funneling erc20 tokens to the TokenizedAccountNFT contract
// TODO: Consider a more efficient implementation based on use, this is a simple O(n^2) approach
// or consider restricting the number of tokens transferrable in a single MsgMicrotx
func FindThresholdIntersection(thresholds []TokenizedAccountThreshold, changedErc20s []*common.Address) []TokenizedAccountThreshold {
	var ret []TokenizedAccountThreshold
	for _, threshold := range thresholds {
		for _, erc20 := range changedErc20s {
			if threshold.Token == *erc20 {
				ret = append(ret, threshold)
			}
		}
	}

	return ret
}
