//SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.19; // Force solidity compliance

import "@openzeppelin/contracts/utils/Context.sol";
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";

/**
 * An abstract contract which provides onlyOwner(id) and onlyOwnerOrApproved(id) modifiers derived from ERC721's
 * onwerOf, getApproved, and isApprovedForAll functions
 *
 * @dev For contracts with manually enumerated tokens (e.g. RockId = 1, PaperId = 2, ScissorsId = 3) use the modifiers like so:
 * ```
 *      // Only the owner of RockId gets to use this function, regardless of approval status
 *      function throwRock() public onlyOwner(RockId) { ... }
 *
 *      // Either the owner, a sender approved for all the tokens the owner of PaperId, or a sender approved for specifically PaperId gets to call this
 *      function wrapWithPaper() public onlyOwnerOrApproved(PaperId) { ... }
 * ```
 */
abstract contract OwnableApprovableERC721 is Context, ERC721 {
    /**
     * @dev Throws if called by any account other than the owner.
     */
    modifier onlyOwner(uint256 tokenId) {
        require(
            ERC721(this).ownerOf(tokenId) == _msgSender(),
            "OwnableApprovable: caller is not the owner"
        );
        _;
    }

    /**
     * @dev Throws if called by any account other than the owner or someone approved by the owner
     */
    modifier onlyOwnerOrApproved(uint256 tokenId) {
        // Get approval directly from ERC721's internal method
        if (_isApprovedOrOwner(_msgSender(), tokenId)) {
            _;
        } else {
            revert("OwnableApprovable: caller is not owner nor approved");
        }
    }
}
