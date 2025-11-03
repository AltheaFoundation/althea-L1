pragma solidity ^0.8.0;

/**
 * @title GovSpendTest
 * @notice A simple contract that can receive Ether and transfer it out
 * @dev This contract can receive Ether via the receive() function without calldata
 */
contract GovSpendTest {
    // Event emitted when Ether is received
    event Received(address indexed sender, uint256 amount);

    // Event emitted when Ether is withdrawn
    event Withdrawn(address indexed recipient, uint256 amount);

    /**
     * @notice Receive function to accept Ether without calldata
     * @dev This is called when Ether is sent to the contract with empty calldata
     */
    receive() external payable {
        emit Received(msg.sender, msg.value);
    }

    /**
     * @notice Fallback function to accept Ether with calldata
     * @dev This is called when Ether is sent with calldata that doesn't match any function
     */
    fallback() external payable {
        emit Received(msg.sender, msg.value);
    }

    /**
     * @notice Get the current Ether balance of this contract
     * @return The balance in wei
     */
    function getBalance() public view returns (uint256) {
        return address(this).balance;
    }

    /**
     * @notice Transfer all Ether from this contract to a recipient (permissionless)
     * @param recipient The address to receive the Ether
     * @return success Whether the transfer succeeded
     */
    function withdrawAll(address payable recipient) public returns (bool success) {
        require(recipient != address(0), "Cannot withdraw to zero address");

        uint256 balance = address(this).balance;
        require(balance > 0, "No balance to withdraw");

        emit Withdrawn(recipient, balance);

        (success, ) = recipient.call{value: balance}("");
        require(success, "Transfer failed");
    }
}
