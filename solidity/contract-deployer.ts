import { WALTHEA } from "./typechain/WALTHEA";
import { TestERC20A } from "./typechain/TestERC20A";
import { TestERC20B } from "./typechain/TestERC20B";
import { TestERC20C } from "./typechain/TestERC20C";
import { TestERC721A } from "./typechain/TestERC721A";
import { GovSpendTest } from "./typechain/GovSpendTest";
import { ethers } from "ethers";
import fs from "fs";
import commandLineArgs from "command-line-args";
import { exit } from "process";

const args = commandLineArgs([
  // the ethernum node used to deploy the contract
  { name: "eth-node", type: String },
  // the Ethereum private key that will contain the gas required to pay for the contact deployment
  { name: "eth-privkey", type: String },
  // path to the artifacts folder
  { name: "artifacts-root", type: String },
]);

// sets the gas price for all contract deployments
const overrides = {
  //gasPrice: 100000000000
}

async function deploy() {
  var startTime = new Date();
  const provider = await new ethers.providers.JsonRpcProvider(args["eth-node"]);
  let wallet = new ethers.Wallet(args["eth-privkey"], provider);
  let artifacts = args["artifacts-root"];

  var success = false;
  while (!success) {
    var present = new Date();
    var timeDiff: number = present.getTime() - startTime.getTime();
    timeDiff = timeDiff / 1000
    provider.getBlockNumber().then(_ => success = true).catch(_ => console.log("Ethereum RPC error, trying again"))

    if (timeDiff > 600) {
      console.log("Could not contact Ethereum RPC after 10 minutes, check the URL!")
      exit(1)
    }
    await sleep(1000);
  }

  console.log("Deploying ERC20 contracts");

  // this handles several possible locations for the ERC20 artifacts
  var erc20_a_path: string = artifacts + "/artifacts/contracts/TestERC20A.sol/TestERC20A.json"
  var erc20_b_path: string = artifacts + "/artifacts/contracts/TestERC20B.sol/TestERC20B.json"
  var erc20_c_path: string = artifacts + "/artifacts/contracts/TestERC20C.sol/TestERC20C.json"
  var erc721_a_path: string = artifacts + "/artifacts/contracts/TestERC721A.sol/TestERC721A.json"
  var walthea_path: string = artifacts + "/artifacts/contracts/WALTHEA.sol/WALTHEA.json"
  var govspendtest_path: string = artifacts + "/artifacts/contracts/GovSpendTest.sol/GovSpendTest.json"

  if (!fs.existsSync(artifacts)) {
    console.log("Artifacts folder not found, please specify the correct path using the --artifacts-root flag")
    exit(1)
  }

  const { abi, bytecode } = getContractArtifacts(erc20_a_path);
  const erc20Factory = new ethers.ContractFactory(abi, bytecode, wallet);
  const testERC20 = (await erc20Factory.deploy(overrides)) as TestERC20A;

  const { abi: abi1, bytecode: bytecode1 } = getContractArtifacts(erc20_b_path);
  const erc20Factory1 = new ethers.ContractFactory(abi1, bytecode1, wallet);
  const testERC201 = (await erc20Factory1.deploy(overrides)) as TestERC20B;

  const { abi: abi2, bytecode: bytecode2 } = getContractArtifacts(erc20_c_path);
  const erc20Factory2 = new ethers.ContractFactory(abi2, bytecode2, wallet);
  const testERC202 = (await erc20Factory2.deploy(overrides)) as TestERC20C;

  const { abi: abi3, bytecode: bytecode3 } = getContractArtifacts(erc721_a_path);
  const erc721Factory1 = new ethers.ContractFactory(abi3, bytecode3, wallet);
  const testERC721 = (await erc721Factory1.deploy(overrides)) as TestERC721A;

  const { abi: abi4, bytecode: bytecode4 } = getContractArtifacts(walthea_path);
  const waltheaFactory = new ethers.ContractFactory(abi4, bytecode4, wallet);
  const walthea = (await waltheaFactory.deploy(overrides)) as WALTHEA;

  const { abi: abi5, bytecode: bytecode5 } = getContractArtifacts(govspendtest_path);
  const govSpendTestFactory = new ethers.ContractFactory(abi5, bytecode5, wallet);
  const govSpendTest = (await govSpendTestFactory.deploy(overrides)) as GovSpendTest;

  // Wait for all deployments to complete in parallel
  console.log("Waiting for all contracts to be deployed...");
  await Promise.all([
    testERC20.deployed(),
    testERC201.deployed(),
    testERC202.deployed(),
    testERC721.deployed(),
    walthea.deployed(),
    govSpendTest.deployed(),
  ]);

  // Log all addresses
  console.log("ERC20 deployed at Address - ", testERC20.address);
  console.log("ERC20 deployed at Address - ", testERC201.address);
  console.log("ERC20 deployed at Address - ", testERC202.address);
  console.log("ERC721 deployed at Address - ", testERC721.address);
  console.log("WALTHEA deployed at Address - ", walthea.address);
  console.log("GovSpendTest deployed at Address - ", govSpendTest.address);
}

function getContractArtifacts(path: string): { bytecode: string; abi: string } {
  var { bytecode, abi } = JSON.parse(fs.readFileSync(path, "utf8").toString());
  return { bytecode, abi };
}

async function main() {
  await deploy();
}

function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

main();
