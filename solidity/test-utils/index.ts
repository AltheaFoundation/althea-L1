import { Gravity } from "../typechain/Gravity";
import { TestERC20A } from "../typechain/TestERC20A";
import { ethers } from "hardhat";
import { makeCheckpoint, getSignerAddresses, ZeroAddress } from "./pure";
import { Signer } from "ethers";

type DeployContractsOptions = {
  corruptSig?: boolean;
};

export async function deployContracts(
  gravityId: string = "foo",
  validators: Signer[],
  powers: number[],
  opts?: DeployContractsOptions
) {
  // enable automining for these tests
  await ethers.provider.send("evm_setAutomine", [true]);

  const TestERC20 = await ethers.getContractFactory("TestERC20A");
  const testERC20 = (await TestERC20.deploy()) as TestERC20A;

  const Gravity = await ethers.getContractFactory("Gravity");

  const valAddresses = await getSignerAddresses(validators);

  const checkpoint = makeCheckpoint(valAddresses, powers, 0, 0, ZeroAddress, gravityId);

  const gravity = (await Gravity.deploy(
    gravityId,
    await getSignerAddresses(validators),
    powers,
  )) as Gravity;

  await gravity.deployed();

  return { gravity, testERC20, checkpoint };
}
