// SPDX-License-Identifier: GPL-3.0

// Copyright (C) 2015, 2016, 2017 Dapphub

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

pragma solidity 0.8.28;

import {ERC20Permit} from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Permit.sol";
import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/// This contract is based on the WETH9 contract found at https://github.com/gnosis/canonical-weth/blob/master/contracts/WETH9.sol
/// with the following changes:
///    name and symbol,
///    solidity compiler version updated to 0.8.28,
///    reimplementation of core ERC20 functions via openzeppelin's contracts
///    addition of the permit() function via openzeppelin's ERC20Permit contract
contract WALTHEA is ERC20Permit {
    constructor() ERC20("Wrapped Althea", "WALTHEA") ERC20Permit("WALTHEA") {}

    event Deposit(address indexed dst, uint indexed amount);
    event Withdrawal(address indexed src, uint indexed amount);

    fallback() external payable {
        deposit();
    }

    receive() external payable {
        deposit();
    }

    function deposit() public payable {
        address sender = _msgSender();
        _mint(sender, msg.value);
        emit Deposit(msg.sender, msg.value);
    }

    function withdraw(uint amount) public {
        address sender = _msgSender();
        _burn(sender, amount);
        payable(sender).transfer(amount);
        emit Withdrawal(msg.sender, amount);
    }
}