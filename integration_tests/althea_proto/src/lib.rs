//! This crate provides althea-chain proto definitions in Rust and also re-exports cosmos_sdk_proto for use by downstream
//! crates. By default around a dozen proto files are generated and placed into the prost folder. We could then proceed
//! to fix up all these files and use them as the required dependencies for the proto files, but we chose instead to replace
//! those paths with references ot upstream cosmos-sdk-proto and delete the other files. This reduces cruft in this repo even
//! if it does make for a somewhat more confusing proto generation process.

pub use cosmos_sdk_proto;
pub mod lockup {
    pub mod v1 {
        include!("prost/lockup.v1.rs");
    }
}
pub mod microtx {
    pub mod v1 {
        include!("prost/microtx.v1.rs");
    }
}

// THIRD PARTY PROTOS MANAGED IN THIS REPO
pub mod canto {
    pub mod csr {
        pub mod v1 {
            include!("prost/canto.csr.v1.rs");
        }
    }
    pub mod epochs {
        pub mod v1 {
            include!("prost/canto.epochs.v1.rs");
        }
    }
    pub mod erc20 {
        pub mod v1 {
            include!("prost/canto.erc20.v1.rs");
        }
    }
    pub mod fees {
        pub mod v1 {
            include!("prost/canto.fees.v1.rs");
        }
    }
    pub mod govshuttle {
        pub mod v1 {
            include!("prost/canto.govshuttle.v1.rs");
        }
    }
    pub mod inflation {
        pub mod v1 {
            include!("prost/canto.inflation.v1.rs");
        }
    }
    pub mod recovery {
        pub mod v1 {
            include!("prost/canto.recovery.v1.rs");
        }
    }
    pub mod vesting {
        pub mod v1 {
            include!("prost/canto.vesting.v1.rs");
        }
    }
}

pub mod ethermint {
    pub mod crypto {
        pub mod v1 {
            pub mod ethsecp256k1 {
                include!("prost/ethermint.crypto.v1.ethsecp256k1.rs");
            }
        }
    }
    pub mod evm {
        pub mod v1 {
            include!("prost/ethermint.evm.v1.rs");
        }
    }
    pub mod feemarket {
        pub mod v1 {
            include!("prost/ethermint.feemarket.v1.rs");
        }
    }
    pub mod types {
        pub mod v1 {
            include!("prost/ethermint.types.v1.rs");
        }
    }
}
