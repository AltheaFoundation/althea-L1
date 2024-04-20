package contracts

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	Uint8Type, u8err    = abi.NewType("uint8", "", nil)
	Uint16Type, u16err  = abi.NewType("uint16", "", nil)
	Uint32Type, u32err  = abi.NewType("uint32", "", nil)
	Uint64Type, u64err  = abi.NewType("uint64", "", nil)
	AddressType, adrerr = abi.NewType("address", "", nil)
	BoolType, boolerr   = abi.NewType("bool", "", nil)
)

var typeMap map[string]abi.Type = map[string]abi.Type{}

func init() {
	if u8err != nil || u16err != nil || u32err != nil || u64err != nil || adrerr != nil || boolerr != nil {
		panic(fmt.Sprintf("failed to create ABI types: %v, %v, %v, %v, %v, %v", u8err, u16err, u32err, u64err, adrerr, boolerr))
	}
	typeMap["bool"] = BoolType
	typeMap["uint8"] = Uint8Type
	typeMap["uint16"] = Uint16Type
	typeMap["uint32"] = Uint32Type
	typeMap["uint64"] = Uint64Type
	typeMap["address"] = AddressType
}

func GetType(name string) (abi.Type, error) {
	if t, ok := typeMap[name]; ok {
		return t, nil
	}
	return abi.Type{}, fmt.Errorf("unknown type %s", name)
}

func GetTypeArguments(names []string) ([]abi.Argument, error) {
	args := make([]abi.Argument, len(names))
	for i, name := range names {
		t, err := GetType(name)
		if err != nil {
			return nil, err
		}
		// nolint: exhaustruct
		args[i] = abi.Argument{Type: t, Name: ""}
	}
	return args, nil
}

func EncodeArguments(args []abi.Argument, values []interface{}) ([]byte, error) {
	return abi.Arguments(args).Pack(values...)
}

func EncodeTypes(names []string, values []interface{}) ([]byte, error) {
	args, err := GetTypeArguments(names)
	if err != nil {
		return nil, err
	}
	return EncodeArguments(args, values)
}
