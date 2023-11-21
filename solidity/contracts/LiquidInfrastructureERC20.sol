//SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.12; // Force solidity compliance

import "@openzeppelin/contracts/utils/math/Math.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import "@openzeppelin/contracts/token/ERC721/utils/ERC721Holder.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";
import "./LiquidInfrastructureNFT.sol";

/**
 * @title Liquid Infrastructure ERC20
 * @author Christian Borst <christian@althea.systems>
 *
 * @dev An ERC20 contract used to earn rewards from managed LiquidInfrastructreNFTs.
 *
 * A LiquidInfrastructureNFT typically represents some form of infrastructure involved in an Althea pay-per-forward network
 * which frequently receives payments from peers on the network for performing an automated service (e.g. providing internet).
 * This LiquidInfrastructureERC20 acts as a convenient aggregation layer to enable dead-simple investment in real-world assets
 * with automatic revenue accrual. Simply by holding this ERC20 owners are entitled to revenue from the network represented by the token.
 *
 * Revenue is gathered from managed LiquidInfrastructureNFTs by the protocol and distributed to token holders on a semi-regular basis,
 * where there is a minimum number of blocks required to elapse before a new payout to token holders.
 */
contract LiquidInfrastructureERC20 is
    ERC20,
    ERC20Burnable,
    Ownable,
    ERC721Holder
{
    event DistributionStarted();
    event Distribution(address recipient);
    event DistributionFinished();
    event WithdrawalStarted();
    event Withdrawal(address source);
    event WithdrawalFinished();

    IERC20[] private distributableERC20s;
    uint256[] private erc20EntitlementPerUnit;
    address[] private holders;

    /**
     * @notice This is the current version of the contract. Every update to the contract will introduce a new
     * version, regardless of anticipated compatibility.
     */
    uint256 public constant Version = 1;

    /**
     * @notice This collection holds the managed LiquidInfrastructureNFTs which periodically generate revenue and deliver
     * the balances to this contract.
     */
    address[] public ManagedNFTs;

    /**
     * @notice Holds the block of the last distribution, used for limiting distribution lock ups
     */
    uint256 public LastDistribution;

    /**
     * @notice Holds the minimum number of blocks required to elapse before a new distribution can begin
     */
    uint256 public MinDistributionPeriod;

    /**
     * @notice When true, locks all transfers, mints, and burns until the current distribution has completed
     */
    bool public LockedForDistribution;

    /**
     * @dev Holds the index into `holders` of the next account owed the current distribution
     */
    uint256 internal nextDistributionRecipient;

    /**
     * @dev Holds the index into `ManagedNFTs` of the next contract to withdraw funds from
     */
    uint256 private nextWithdrawal;

    /**
     * Implements the lock during distributions, adds `to` to the list of holders when needed
     * @param from token sender
     * @param to  token receiver
     * @param amount  amount sent
     */
    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal virtual override {
        require(!LockedForDistribution, "distribution in progress");
        if (from == address(0)) {
            _beforeMint(to, amount);
        }
        if (to == address(0)) {
            _beforeBurn(from, amount);
        }
        bool exists = (this.balanceOf(to) == 0);
        if (!exists) {
            holders.push(to);
        }
    }

    /**
     * TODO: Reevaluate - maybe this should be combined with _beforeBurn()
     *
     * Implements an additional lock on minting, ensuring that mints happen after any potential distributions
     * @param to the receiver of minted tokens
     * @param amount the amount minted
     */
    function _beforeMint(address to, uint256 amount) internal view {
        require(
            !_isPastMinDistributionPeriod(),
            "must distribute before minting"
        );
    }

    /**
     * TODO: Reevaluate - maybe this should be combined with _beforeMint()
     *
     * Implements an additional lock on burning, ensuring that burns happen after any potential distributions
     * @param to the receiver of minted tokens
     * @param amount the amount minted
     */
    function _beforeBurn(address to, uint256 amount) internal view {
        require(
            !_isPastMinDistributionPeriod(),
            "must distribute before burning"
        );
    }

    /**
     * Removes `from` from the list of holders when they no longer hold any balance
     * @param from token sender
     * @param to  token receiver
     * @param amount  amount sent
     */
    function _afterTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal virtual override {
        bool stillHolding = (this.balanceOf(from) == 0);
        if (!stillHolding) {
            for (uint i = 0; i < holders.length; i++) {
                if (holders[i] == from) {
                    // Remove the element at i by copying the last one into its place and removing the last element
                    holders[i] = holders[holders.length - 1];
                    holders.pop();
                }
            }
        }
    }

    /**
     * Begins or continues a distribution, preventing transfers, mints, and burns of the token until all rewards have been paid out
     *
     * @notice distributions may only begin once every MinDistributionPeriod.
     *
     * @param numDistributions the number of distributions to process in this execution
     */
    function distribute(uint256 numDistributions) public {
        require(numDistributions > 0, "must process at least 1 distribution");
        if (!LockedForDistribution) {
            require(
                _isPastMinDistributionPeriod(),
                "MinDistributionPeriod not met"
            );
            _beginDistribution();
        }

        uint256 limit = Math.min(
            nextDistributionRecipient + numDistributions,
            holders.length
        );

        uint i;
        for (i = nextDistributionRecipient; i < limit; i++) {
            address recipient = holders[i];
            for (uint j = 0; j < distributableERC20s.length; j++) {
                IERC20 toDistribute = IERC20(distributableERC20s[j]);
                uint256 entitlement = erc20EntitlementPerUnit[j] *
                    this.balanceOf(recipient);
                bool success = toDistribute.transferFrom(
                    address(this),
                    recipient,
                    entitlement
                );
                require(success, "failed to distribute to recipient");
            }
            emit Distribution(recipient);
        }
        nextDistributionRecipient = i + 1;

        if (nextDistributionRecipient == holders.length) {
            _endDistribution();
        }
    }

    function _isPastMinDistributionPeriod() internal view returns (bool) {
        return (block.number - LastDistribution) >= MinDistributionPeriod;
    }

    /**
     * Prepares this contract for distribution:
     * - Locks the contract
     * - Calculates the entitlement to protocol-held ERC20s per unit of the LiquidInfrastructureERC20 held
     */
    function _beginDistribution() internal {
        LockedForDistribution = true;

        // clear the previous entitlements, if any
        if (erc20EntitlementPerUnit.length > 0) {
            delete erc20EntitlementPerUnit;
        }

        // Calculate the entitlement per token held
        uint256 supply = this.totalSupply();
        for (uint i = 0; i < distributableERC20s.length; i++) {
            uint256 entitlement = IERC20(distributableERC20s[i]).balanceOf(
                address(this)
            ) / supply;
            erc20EntitlementPerUnit.push(entitlement);
        }

        nextDistributionRecipient = 0;
        emit DistributionStarted();
    }

    /**
     * Unlocks this contract at the end of a distribution
     */
    function _endDistribution() internal {
        delete erc20EntitlementPerUnit;
        LockedForDistribution = false;
        emit DistributionFinished();
    }

    /**
     * Convenience function that allows the contract owner to distribute when necessary and then mint right after
     *
     * @notice attempts to distribute to every holder in this block, which may exceed the block gas limit
     * if this fails then first call distribute
     */
    function mintAndDistribute(
        address account,
        uint256 amount
    ) public onlyOwner {
        if (_isPastMinDistributionPeriod()) {
            distribute(holders.length);
        }
        mint(account, amount);
    }

    /**
     * Allows the contract owner to mint tokens for an address
     *
     * @notice minting may only occur when a distribution has happened within MinDistributionPeriod blocks
     */
    function mint(address account, uint256 amount) public onlyOwner {
        _mint(account, amount);
    }

    /**
     * Convenience function that allows a token holder to distribute when necessary and then burn their tokens right after
     *
     * @notice attempts to distribute to every holder in this block, which may exceed the block gas limit
     * if this fails then first call distribute() enough times to finish a distribution and then call burn()
     */
    function burnAndDistribute(uint256 amount) public {
        if (_isPastMinDistributionPeriod()) {
            distribute(holders.length);
        }
        burn(amount);
    }

    /**
     * Convenience function that allows an approved sender to distribute when necessary and then burn the approved tokens right after
     *
     * @notice attempts to distribute to every holder in this block, which may exceed the block gas limit
     * if this fails then first call distribute() enough times to finish a distribution and then call burnFrom()
     */
    function burnFromAndDistribute(address account, uint256 amount) public {
        if (_isPastMinDistributionPeriod()) {
            distribute(holders.length);
        }
        burnFrom(account, amount);
    }

    /**
     * Performs withdrawals from the ManagedNFTs collection, depositing all token balances into the custody of this contract
     * @param numWithdrawals the number of withdrawals to perform
     */
    function withdrawFromManagedNFTs(uint256 numWithdrawals) public {
        require(!LockedForDistribution, "cannot withdraw during distribution");

        if (nextWithdrawal == 0) {
            emit WithdrawalStarted();
        }

        uint256 limit = Math.min(
            numWithdrawals + nextWithdrawal,
            ManagedNFTs.length
        );
        uint256 i;
        for (i = nextWithdrawal; i < limit; i++) {
            LiquidInfrastructureNFT withdrawFrom = LiquidInfrastructureNFT(
                ManagedNFTs[i]
            );

            (address[] memory withdrawERC20s, ) = withdrawFrom.getThresholds();
            withdrawFrom.withdrawBalancesTo(withdrawERC20s, address(this));
            emit Withdrawal(address(withdrawFrom));
        }
        nextWithdrawal = i + 1;

        if (nextWithdrawal == ManagedNFTs.length) {
            nextWithdrawal = 0;
            emit WithdrawalFinished();
        }
    }

    /**
     * Constructs the underlying ERC20 and initializes critical variables
     *
     * @param _managedNFTs The addresses of the controlled LiquidInfrastructureNFT contracts
     */
    constructor(
        string memory _name,
        string memory _symbol,
        address[] memory _managedNFTs
    ) ERC20(_name, _symbol) Ownable() {
        ManagedNFTs = _managedNFTs;
        LastDistribution = block.number;
    }
}
