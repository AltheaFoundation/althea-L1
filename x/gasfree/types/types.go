package types

const (
	// ModuleName is the name of the module
	ModuleName = "gasfree"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName
)

var (
	// GasFreeMessageTypesKey Indexes the GasFreeMessageTypes array, the collection of messages which
	// will NOT be charged gas immediately when they execute, but must define an alternate gas payment
	// method in their Msg handler
	GasFreeMessageTypesKey = []byte("gasFreeMessageTypes")

	// GasFreeErc20InteropTokens is a list of ERC20 addresses in the EVM or Cosmos denoms in the bank module
	// which can be used to pay for gas for specifically the erc20 module's gasfree messages.
	// The erc20 module's gasfree messages are MsgSendCoinToEVM, MsgSendERC20ToCosmos, and MsgSendERC20ToCosmosAndIBCTransfer
	GasFreeErc20InteropTokensKey = []byte("gasFreeErc20InteropTokens")

	// GasFreeErc20InteropFeeBasisPoints specifies the percentage fee taken from the user who submits one of the
	// erc20 module's gasfree messages. The fee is a percentage of the amount of tokens converted in the message.
	// The erc20 module's gasfree messages are MsgSendCoinToEVM, MsgSendERC20ToCosmos, and MsgSendERC20ToCosmosAndIBCTransfer
	GasFreeErc20InteropFeeBasisPointsKey = []byte("gasFreeErc20InteropFeeBasisPoints")
)
