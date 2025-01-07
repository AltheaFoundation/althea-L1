//SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.28; // Force solidity compliance
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

// One of three testing coins
contract TestERC20B is ERC20 {
    constructor() ERC20("2 Ethereum", "E2H") {
        _mint(
            0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,
            100000000000000000000000000
        );
        _mint(
            0x37f0C93ae1b73d0F8DbCcEa1514d1434a8D70B3b,
            100000000000000000000000000
        );
        _mint(
            0x653E44056b6312Db41b31B25c301853e67c5e8C7,
            100000000000000000000000000
        );
        _mint(
            0xc09B3C4F32D9b2A4D9dE17afCb6b16BD10fE8E61,
            100000000000000000000000000
        );
        _mint(
            0xf21855F6438B591a5dfDBd3DB32D0502b21d8349,
            100000000000000000000000000
        );
        _mint(
            0xc09B3C4F32D9b2A4D9dE17afCb6b16BD10fE8E61,
            100000000000000000000000000
        );
        // this is the EtherBase address for our testnet miner in
        // tests/assets/ETHGenesis.json so it wil have both a lot
        // of ETH and a lot of erc20 tokens to test with
        _mint(
            0xBf660843528035a5A4921534E156a27e64B231fE,
            100000000000000000000000000
        );
    }
}
