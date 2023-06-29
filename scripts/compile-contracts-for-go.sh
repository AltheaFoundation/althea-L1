ROOT="$(dirname "$0")/.."
CONTRACTS_GO_OUTPUT="${ROOT}/contracts/compiled"
FULL_CONTRACTS_ROOT="${ROOT}/solidity"
FULL_CONTRACTS_ARTIFACTS="${FULL_CONTRACTS_ROOT}/artifacts/contracts"
FULL_CONTRACTS_SOURCE="${FULL_CONTRACTS_ROOT}/contracts"

for f in $(ls ${FULL_CONTRACTS_ARTIFACTS}/ | awk '!/Test/') ; do
    # Uncomment to copy the source file to the contracts directory
    # sourceFile="${FULL_CONTRACTS_SOURCE}/$f"
    # sourceCopy="${CONTRACTS_GO_OUTPUT}/$f"
    # cp $sourceFile $sourceCopy

    # Make the contracts/compiled directory
    mkdir -p $CONTRACTS_GO_OUTPUT

    # Get the compiled JSON in solidity/artifacts/contracts/*.sol/*.json, format and output at contracts/compiled/
    compiledFile="${FULL_CONTRACTS_ARTIFACTS}/$f/${f%.sol}.json"
    outputFile="${CONTRACTS_GO_OUTPUT}/${f%.sol}.json"
    echo "Formatting JSON at $compiledFile and copying to $outputFile"
    # Get the bytecode and strip the leading 0x prefix:
    BYTECODE=$(cat $compiledFile | jq -r '.bytecode')
    BYTECODE=${BYTECODE#0x}

    # We need to escape the ABI input, which jq can do for us if we first isolate it
    ABI=$(cat $compiledFile | jq -c '.abi' )
    # and then pass it as a variable for use in the output
    cat $compiledFile | jq --arg abi_escaped ${ABI} --arg bytecode ${BYTECODE} '{ contractName: .contractName, abi: $abi_escaped, bin: $bytecode }' > $outputFile
done