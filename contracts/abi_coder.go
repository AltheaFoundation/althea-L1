package contracts

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	Uint8Type, _   = abi.NewType("uint8", "", nil)
	Uint16Type, _  = abi.NewType("uint16", "", nil)
	Uint32Type, _  = abi.NewType("uint32", "", nil)
	Uint64Type, _  = abi.NewType("uint64", "", nil)
	AddressType, _ = abi.NewType("address", "", nil)
	BoolType, _    = abi.NewType("bool", "", nil)
)

var typeMap map[string]abi.Type = map[string]abi.Type{}

func init() {
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
