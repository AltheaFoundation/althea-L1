//SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.19; // Force solidity compliance

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "./OwnableApprovableERC721.sol";

/**
 * @title Liquid Infrastructure NFT
 * @author Christian Borst <christian@althea.systems>
 *
 * @dev An NFT contract used to control a Liquid Infrastructure Account - a Cosmos Bank module account intrinsically connected to the EVM
 * through Althea's x/microtx module. On chains which are not Althea-L1 this is a standalone ERC721 which should receive revenue periodically,
 * and the protocol manager must ensure LiquidInfrastructureNFTs receive the revenue they are owed.
 *
 * A Liquid Infrastructure Account typically represents some form of infrastructure involved in an Althea pay-per-forward network
 * which frequently receives payments from peers on the network for performing an automated service (e.g. providing internet).
 * Each instance of this LiquidInfrastructureNFT contract represents one x/bank module account, the address of which is a part of
 * the ERC1155 URI.
 *
 * As the x/microtx module is used to conduct microtransactions (the payment layer for Althea networks), receiving accounts
 * will accrue Cosmos Coins and likely spend these same tokens to pay their upstream costs. If any such account is a Liquid
 * account (see x/microtx documentation), the x/microtx module will query this LiquidInfrastructureNFT's BalanceThresholds
 * values to determine how much of which tokens to leave in the x/bank account. All excess amounts will be converted
 * from Cosmos Coins to ERC20s and deposited here. The owner of the Liquid Account may later withdraw the balances
 * this NFT holds via the withdrawBalances() function.
 *
 * Occassionally devices and wallets can be lost, in which case the owner of this contract can call recoverAccount()
 * to begin a recovery process which will finish after the transaction completes. Asynchronously the x/microtx module
 * will ignore thresholds and transfer all of the Liquid Account's balances to this NFT, which may be withdrawn
 * normally with withdrawBalances().
 */
contract LiquidInfrastructureNFT is ERC721, OwnableApprovableERC721 {
    event SuccessfulWithdrawal(address to, address[] erc20s, uint256[] amounts);
    event TryRecover();
    event SuccessfulRecovery(address[] erc20s, uint256[] amounts);
    event ThresholdsChanged(address[] newErc20s, uint256[] newAmounts);

    address[] private thresholdErc20s;
    uint256[] private thresholdAmounts;

    /**
     * @notice This is the current version of the contract. Every update to the contract will introduce a new
     * version, regardless of anticipated compatibility.
     */
    uint256 public constant Version = 1;

    /**
     * @notice This NFT holds only 1 token in it, which is the Account token. Its Id is `1`.
     * This Id is used for access control via the onlyOwner/onlyOwnerOrApproved modifiers.
     * See OwnableApprovableERC721 for more info.
     */
    uint256 public constant AccountId = 1;

    /**
     * Constructs the underlying ERC721 with a URI like "althea://liquid-infrastructure-account/{accountName}", and
     * a symbol like "LIA:{accountName}".
     * Mints the Account token (ID=1), the only token held in this NFT.
     *
     * @param accountName The bech32 address of the controlled x/bank account
     */
    constructor(
        string memory accountName
    )
        ERC721(
            string.concat(
                "althea://liquid-infrastructure-account/",
                accountName
            ),
            string.concat("LIA:", accountName)
        )
    {
        _mint(msg.sender, AccountId);
    }

    /**
     * @dev Returns the current thresholds as a collection of ERC20 addresses and a collection of balance thresholds.
     * These thresholds will be used by the x/microtx module to control the operating balances that the
     * liquid account is allowed to hold. Anything in excess of these balances are transferred to this contract.
     *
     * @return address[]: The ERC20 balances to control
     * @return uint256[]: The maximum operating amount of the associated ERC20
     */
    function getThresholds()
        public
        view
        virtual
        returns (address[] memory, uint256[] memory)
    {
        return (thresholdErc20s, thresholdAmounts);
    }

    /**
     * @dev Updates the threshold values used by the x/microtx module which determine precisely how much
     * of each token should be left in the Liquid Account's x/bank account.
     *
     * Any excess balances will accumulate here, and may be retrieved by the owner with
     * the withdrawBalances(erc20s) method
     *
     * The ERC20s specified here have EVM addressses, querying the x/erc20 module will reveal their
     * Cosmos Coin counterparts
     *
     * @param newErc20s The new threshold addresses to set
     * @param newAmounts The new threshold amounts to set
     *
     * @notice this function is access controlled, only the owner or an approved msg.sender may call this function
     */
    function setThresholds(
        address[] calldata newErc20s,
        uint256[] calldata newAmounts
    ) public virtual onlyOwnerOrApproved(AccountId) {
        require(
            newErc20s.length == newAmounts.length,
            "threshold values must have the same length"
        );
        // Clear the thresholds before overwriting
        delete thresholdErc20s;
        delete thresholdAmounts;

        for (uint i = 0; i < newErc20s.length; i++) {
            thresholdErc20s.push(newErc20s[i]);
            thresholdAmounts.push(newAmounts[i]);
        }
        emit ThresholdsChanged(newErc20s, newAmounts);
    }

    /**
     * @dev Withdraws ERC20 balances from this contract to the owner, throws on any erc20 transfer() failure
     * Emits a {SuccessfulWithdrawal} event upon success.
     *
     * @param erc20s A list of ERC20 tokens to withdraw NFT balances from
     *
     * @notice This function is access controlled, only the owner or an approved msg.sender may call this function
     */
    function withdrawBalances(address[] calldata erc20s) public virtual {
        require(
            _isApprovedOrOwner(_msgSender(), AccountId),
            "caller is not the owner of the Account token and is not approved either"
        );
        address destination = ownerOf(AccountId);
        _withdrawBalancesTo(erc20s, destination);
    }

    /**
     * @dev Withdraws ERC20 balances from this contract to `destination`, throws on any erc20 transfer() failure
     * Emits a {SuccessfulWithdrawal} event upon success.
     *
     * @param erc20s A list of ERC20 tokens to withdraw NFT balances from
     * @param destination The address to send all the NFT's ERC20 balances to
     *
     * @notice This function is access controlled, only the owner or an approved msg.sender may call this function
     */
    function withdrawBalancesTo(
        address[] calldata erc20s,
        address destination
    ) public virtual {
        require(
            _isApprovedOrOwner(_msgSender(), AccountId),
            "caller is not the owner of the Account token and is not approved either"
        );
        _withdrawBalancesTo(erc20s, destination);
    }

    /**
     * @dev Internal withdrawal used because withdrawBalances cannot call withdrawBalancesTo without error
     *
     * @param erc20s A list of ERC20 tokens to withdraw NFT balances from
     * @param destination The address to send all the NFT's ERC20 balances to
     */
    function _withdrawBalancesTo(
        address[] calldata erc20s,
        address destination
    ) internal {
        uint256[] memory amounts = new uint256[](erc20s.length);
        for (uint i = 0; i < erc20s.length; i++) {
            address erc20 = erc20s[i];
            uint256 balance = IERC20(erc20).balanceOf(address(this));
            if (balance > 0) {
                bool result = IERC20(erc20).transfer(destination, balance);
                require(result, "unsuccessful withdrawal");
                amounts[i] = balance;
            }
        }
        emit SuccessfulWithdrawal(destination, erc20s, amounts);
    }

    /**
     * @dev Begins the Liquid Infrastructure Account recovery process, which must be detected by the x/microtx module.
     * Emits a {TryRecover} event which must be detected by the x/microtx module.
     * Expected behavior is that all tokens held in the x/bank account of the Liquid Account will be
     * converted to ERC20s and transferred to the control of this contract. If the recovery is successful
     * then the x/microtx module will append a {SuccessfulRecovery} event with the ERC20 addresses and amounts
     * sent to this contract.
     *
     * After a successful recovery, use withdrawBalances(erc20s) to send the token balances to the owner account.
     *
     * @notice Due to the EVM <> Cosmos interactions, the tokens will not be transferred to this contract until
     * the entire transaction is resolved. It will not be possible to recoverAccount() and immediately withdrawBalances()
     * in the same transaction!
     *
     * @notice This function is access controlled, only the owner may call this function
     */
    function recoverAccount() public virtual onlyOwner(AccountId) {
        emit TryRecover();
    }
}
