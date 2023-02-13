import { TestERC20A } from "../typechain/TestERC20A";
import { ethers } from "hardhat";

type DeployContractsOptions = {
  corruptSig?: boolean;
};

export async function deployContracts() {

  const TestERC20 = await ethers.getContractFactory("TestERC20A");
  const testERC20 = (await TestERC20.deploy()) as TestERC20A;

  return { testERC20 };
}
