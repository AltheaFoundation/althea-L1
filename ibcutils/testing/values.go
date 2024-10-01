/*
This file contains the variables, constants, and default values
used in the testing package and commonly defined in tests.
*/
package ibctesting

import (
	"time"

	ibctransfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	ibctmtypes "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint/types"
	"github.com/cosmos/ibc-go/v6/testing/mock"
)

const (
	// Default params constants used to create a TM client
	TrustingPeriod     time.Duration = time.Hour * 24 * 7 * 2
	UnbondingPeriod    time.Duration = time.Hour * 24 * 7 * 3
	MaxClockDrift      time.Duration = time.Second * 10
	DefaultDelayPeriod uint64        = 0

	DefaultChannelVersion = mock.Version

	// Application Ports
	TransferPort = ibctransfertypes.ModuleName
)

var (
	DefaultOpenInitVersion *connectiontypes.Version

	// Default params variables used to create a TM client
	DefaultTrustLevel ibctmtypes.Fraction = ibctmtypes.DefaultTrustLevel

	UpgradePath = []string{"upgrade", "upgradedIBCState"}

	ConnectionVersion = connectiontypes.ExportedVersionsToProto(connectiontypes.GetCompatibleVersions())[0]
)
