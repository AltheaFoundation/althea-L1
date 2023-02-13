import { ethers } from "hardhat";
import { Signer } from "ethers";
import { ContractTransaction, utils } from 'ethers';

export const ZeroAddress: string = "0x0000000000000000000000000000000000000000"

export async function getSignerAddresses(signers: Signer[]) {
  return await Promise.all(signers.map(signer => signer.getAddress()));
}

export type Sig = {
  v: number,
  r: string,
  s: string
};

export async function signHash(signers: Signer[], hash: string) {
  let sigs: Sig[] = [];

  for (let i = 0; i < signers.length; i = i + 1) {
    const sig = await signers[i].signMessage(ethers.utils.arrayify(hash));
    const address = await signers[i].getAddress();

    const splitSig = ethers.utils.splitSignature(sig);
    sigs.push({ v: splitSig.v!, r: splitSig.r, s: splitSig.s });
  }

  return sigs;
}

export async function parseEvent(contract: any, txPromise: Promise<ContractTransaction>, eventOrder: number) {
  const tx = await txPromise
  const receipt = await contract.provider.getTransactionReceipt(tx.hash!)
  let args = (contract.interface as utils.Interface).parseLog(receipt.logs![eventOrder]).args

  // Get rid of weird quasi-array keys
  const acc: any = {}
  args = Object.keys(args).reduce((acc, key) => {
    if (Number.isNaN(parseInt(key, 10)) && key !== 'length') {
      acc[key] = args[key]
    }
    return acc
  }, acc)

  return args
}