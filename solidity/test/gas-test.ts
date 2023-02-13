import chai from "chai";
import { ethers } from "hardhat";
import { solidity } from "ethereum-waffle";

import { deployContracts } from "../test-utils";
import {
    getSignerAddresses,
    signHash,
    examplePowers,
    ZeroAddress
} from "../test-utils/pure";
import { BigNumber, BigNumberish } from "ethers";

chai.use(solidity);
const { expect } = chai;

describe("Gas tests", function () {
    it("makeCheckpoint in isolation", async function () {
        const signers = await ethers.getSigners();
        const gravityId = ethers.utils.formatBytes32String("foo");

        // This is the power distribution on the Cosmos hub as of 7/14/2020
        let powers = examplePowers();
        let validators = signers.slice(0, powers.length);

        const {
            gravity,
            testERC20,
            checkpoint: deployCheckpoint
        } = await deployContracts(gravityId, validators, powers);

        let valset = {
            validators: await getSignerAddresses(validators),
            powers,
            valsetNonce: 0,
            rewardAmount: 0,
            rewardToken: ZeroAddress
        }

        await gravity.testMakeCheckpoint(
            valset,
            gravityId
        );
    });

    it("checkValidatorSignatures in isolation", async function () {
        const signers = await ethers.getSigners();
        const gravityId = ethers.utils.formatBytes32String("foo");

        // This is the power distribution on the Cosmos hub as of 7/14/2020
        let powers = examplePowers();
        let validators = signers.slice(0, powers.length);

        const {
            gravity,
            testERC20,
            checkpoint: deployCheckpoint
        } = await deployContracts(gravityId, validators, powers);

        let sigs = await signHash(
            validators,
            "0x7bc422a00c175cae98cf2f4c36f2f8b63ec51ab8c57fecda9bccf0987ae2d67d"
        );


        let v = {
            validators: await getSignerAddresses(validators), powers: powers, valsetNonce: 0, rewardAmount: 0, rewardToken: ZeroAddress
        };
        await gravity.testCheckValidatorSignatures(
            v,
            sigs,
            "0x7bc422a00c175cae98cf2f4c36f2f8b63ec51ab8c57fecda9bccf0987ae2d67d",
            6666
        );
    });
});
