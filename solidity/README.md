# Solidity (a.k.a. Althea Contracts)

This folder is the home of the Solidity contracts deployed by Althea-L1.
The `althea` binary uses the formatted output of these contracts (notably contracts/LiquidInfrastructureNFT.sol) via a go embedding, so to build the binary it is necessary to compile the contents of this folder.

The contract source files live in `contracts/`. The tests live in `test/` and may use `test-utils/` for reusable testing components.
Testing relies on the use of [HardHat](https://hardhat.org/) and `contract-deployer.ts`, which is called via `scripts/contract-deployer.sh`. Contract deployer is used by the integration tests, so be careful not to break it!

## Compiling the contracts

1. Run `npm install`
1. Run `npm run typechain`
1. The compiled files are all placed in artifacts/contracts/\<Contract Name\>.sol/\<Contract Name\>.json, these are directly usable with libraries like ethers.js.

Note that for use with Go there is an additional formatting step, in the project root you can run `make contracts` and observe the output in `contracts/compiled/\<Contract Name\>.json`

## Testing the contracts

The tests should use [Chai](https://www.chaijs.com/) with the [ethereum-waffle extensions](https://ethereum-waffle.readthedocs.io/en/latest/).

Define tests in the `test/` folder and then run `npm run test` to run them.

## Contract Names

It is important to not name a contract which will be used by the chain with any name containing "Test". Any such contract will not be compiled and made into a release artifact, see `.build.sh` in the project root for where that grep-based exclusion happens.