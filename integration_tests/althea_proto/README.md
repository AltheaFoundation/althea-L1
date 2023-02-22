# althea_proto

[![Crate][crate-image]][crate-link]
![Apache 2.0 Licensed][license-image]

Rust crate for interacting with compiled Protobufs used by [Althea Chain].

Because no crate provides neither the upstream [Canto Protobufs] nor [Ethermint Protobufs],
this crate is also responsible for compiling and distributing those. Any protos not prefixed
with an organization name (e.g. src/prost/canto.csr.v1.rs, src/prost/ethermint.types.v1.rs)
are specific to Althea Chain.

This crate also provides the [Cosmos Protobufs] by exporting the [Cosmos SDK Proto] crate,
purely for convenience.

[//]: # "badges"
[crate-image]: https://img.shields.io/crates/v/althea_proto.svg?logo=rust
[crate-link]: https://crates.io/crates/althea_proto
[license-image]: https://img.shields.io/badge/license-Apache2.0-blue.svg

[//]: # "general links"
[Cosmos Protobufs]: https://github.com/cosmos/cosmos-sdk/tree/master/proto/
[Cosmos SDK]: https://github.com/cosmos/cosmos-sdk
[Cosmos SDK Proto]: https://crates.io/crates/cosmos-sdk-proto/
[Althea Chain]: https://github.com/althea-net/althea-chain
[Ethermint Protobufs]: https://github.com/evmos/ethermint/tree/main/proto/
[Canto Protobufs]: https://github.com/Canto-Network/Canto/tree/main/proto/
[Althea Protobufs]: https://github.com/althea-net/althea-chain/tree/main/proto/