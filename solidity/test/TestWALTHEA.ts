import { expect } from "chai";
import { ethers } from "hardhat";
import { SignerWithAddress } from "@nomiclabs/hardhat-ethers/signers";
import { WALTHEA } from "../typechain/WALTHEA"; // Update paths based on your setup

describe("WALTHEA Contract", function () {
  let waltheaFactory: any;
  let walthea: WALTHEA;
  let owner: SignerWithAddress;
  let addr1: SignerWithAddress;
  let addr2: SignerWithAddress;
  let addr3: SignerWithAddress;

  beforeEach(async () => {
    [owner, addr1, addr2, addr3] = await ethers.getSigners();
    waltheaFactory = await ethers.getContractFactory("WALTHEA");
    walthea = (await waltheaFactory.deploy()) as WALTHEA;
    await walthea.deployed();
  });

  it("Should have correct name symbol and decimals", async function () {
    expect(await walthea.name()).to.equal("Wrapped Althea");
    expect(await walthea.symbol()).to.equal("WALTHEA");
    expect(await walthea.decimals()).to.equal(18);
  });

  it("Should deposit native token and mint WALTHEA tokens", async function () {
    // Perform a deposit of 1 ether
    const depositAmount = ethers.utils.parseEther("1");
    const beforeEthBalance = await addr1.getBalance();
    let tx = await walthea.connect(addr1).deposit({ value: depositAmount });
    let receipt = await tx.wait();
    const gasUsed = receipt.effectiveGasPrice.mul(receipt.gasUsed);

    const wrappedBalance = await walthea.balanceOf(addr1.address);
    // Wrapped balance should be 1 ether
    expect(wrappedBalance).to.equal(depositAmount);

    // The final eth balance should be less than the initial balance by the deposit amount + gas used
    const finalEthBalance = await addr1.getBalance();
    const balanceDiff = finalEthBalance.sub(beforeEthBalance).add(gasUsed);
    expect(balanceDiff.abs().sub(depositAmount).abs().lte(ethers.utils.parseEther("0.001"))).to.be.true;  
  });

  it("Should emit Deposit event on deposit", async function () {
    const depositAmount = ethers.utils.parseEther("0.5");
    await expect(walthea.connect(addr2).deposit({ value: depositAmount }))
      .to.emit(walthea, "Deposit")
      .withArgs(addr2.address, depositAmount);
  });

  it("Should withdraw WALTHEA tokens and return native token", async function () {
    // Perform a deposit of 2 ether
    const depositAmount = ethers.utils.parseEther("2");
    await walthea.connect(addr1).deposit({ value: depositAmount });

    const beforeEthBalance = await addr1.getBalance();
    const withdrawTx = await walthea.connect(addr1).withdraw(depositAmount);
    const receipt = await withdrawTx.wait();

    const gasUsed = receipt.effectiveGasPrice.mul(receipt.gasUsed);
    const finalEthBalance = await addr1.getBalance();

    expect(await walthea.balanceOf(addr1.address)).to.equal(0);
    const balanceDiff = finalEthBalance.add(gasUsed).sub(beforeEthBalance);
    // The final eth balance should be more than the initial balance by the deposit amount - gas used
    expect(balanceDiff.sub(depositAmount).abs().lte(ethers.utils.parseEther("0.001"))).to.be.true;
  });

  it("Should emit Withdrawal event on withdraw", async function () {
    const depositAmount = ethers.utils.parseEther("0.3");
    await walthea.connect(addr1).deposit({ value: depositAmount });
    await expect(walthea.connect(addr1).withdraw(depositAmount))
      .to.emit(walthea, "Withdrawal")
      .withArgs(addr1.address, depositAmount);
  });

  it("Fallback and receive should call deposit()", async function () {
    const depositAmount = ethers.utils.parseEther("1");
    await addr2.sendTransaction({
      to: walthea.address,
      value: depositAmount,
    });
    const balance = await walthea.balanceOf(addr2.address);
    expect(balance).to.equal(depositAmount);
  });

  it("Should revert if trying to withdraw more than balance", async function () {
      const depositAmount = ethers.utils.parseEther("1");
      await walthea.connect(addr1).deposit({ value: depositAmount });
      // I am unable to use .to.be.revertedWithCustomError() here because the versions are not working out,
      // it would be worth making a new hardhat project and copying over the generated deps to use that function here
      await expect(walthea.connect(addr1).withdraw(depositAmount.add(1)))
      .to.be.reverted;
  });

  it("Should handle deposits, withdrawals, and transfers correctly", async function () {
    const depositAmount1 = ethers.utils.parseEther("1");
    const depositAmount2 = ethers.utils.parseEther("2");
    const depositAmount3 = ethers.utils.parseEther("3");

    // addr1 deposits
    await expect(walthea.connect(addr1).deposit({ value: depositAmount1 }))
      .to.emit(walthea, "Deposit")
      .withArgs(addr1.address, depositAmount1);
    expect(await walthea.balanceOf(addr1.address)).to.equal(depositAmount1);

    // addr2 deposits
    await expect(walthea.connect(addr2).deposit({ value: depositAmount2 }))
      .to.emit(walthea, "Deposit")
      .withArgs(addr2.address, depositAmount2);
    expect(await walthea.balanceOf(addr2.address)).to.equal(depositAmount2);

    // addr3 deposits
    await expect(walthea.connect(addr3).deposit({ value: depositAmount3 }))
      .to.emit(walthea, "Deposit")
      .withArgs(addr3.address, depositAmount3);
    expect(await walthea.balanceOf(addr3.address)).to.equal(depositAmount3);

    // addr1 withdraws
    const beforeEthBalance1 = await addr1.getBalance();
    const withdrawTx1 = await walthea.connect(addr1).withdraw(depositAmount1);
    const receipt1 = await withdrawTx1.wait();
    const gasUsed1 = receipt1.effectiveGasPrice.mul(receipt1.gasUsed);
    const finalEthBalance1 = await addr1.getBalance();
    expect(await walthea.balanceOf(addr1.address)).to.equal(0);
    expect(finalEthBalance1.add(gasUsed1).sub(beforeEthBalance1).sub(depositAmount1).abs().lte(ethers.utils.parseEther("0.001"))).to.be.true;
    await expect(withdrawTx1).to.emit(walthea, "Withdrawal").withArgs(addr1.address, depositAmount1);

    // addr2 transfers to addr3
    await expect(walthea.connect(addr2).transfer(addr3.address, depositAmount2))
      .to.emit(walthea, "Transfer")
      .withArgs(addr2.address, addr3.address, depositAmount2);
    expect(await walthea.balanceOf(addr2.address)).to.equal(0);
    expect(await walthea.balanceOf(addr3.address)).to.equal(depositAmount3.add(depositAmount2));

    // addr3 withdraws
    const beforeEthBalance3 = await addr3.getBalance();
    const withdrawTx3 = await walthea.connect(addr3).withdraw(depositAmount3.add(depositAmount2));
    const receipt3 = await withdrawTx3.wait();
    const gasUsed3 = receipt3.effectiveGasPrice.mul(receipt3.gasUsed);
    const finalEthBalance3 = await addr3.getBalance();
    expect(await walthea.balanceOf(addr3.address)).to.equal(0);
    expect(finalEthBalance3.add(gasUsed3).sub(beforeEthBalance3).sub(depositAmount3.add(depositAmount2)).abs().lte(ethers.utils.parseEther("0.001"))).to.be.true;
    await expect(withdrawTx3).to.emit(walthea, "Withdrawal").withArgs(addr3.address, depositAmount3.add(depositAmount2));
  });
});