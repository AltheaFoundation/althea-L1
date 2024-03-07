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
)
