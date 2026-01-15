import { useState } from 'react';
import { useAccount } from 'wagmi';
import { formatUnits } from 'viem';
import {
  useContractBalance,
  useUSDCBalance,
  useDeposit,
  useWithdraw,
  useApproveUSDC,
  useMintUSDC,
} from '../hooks/useContract';

export function DepositWithdraw() {
  const { address, isConnected } = useAccount();
  const [amount, setAmount] = useState('100');
  const [mode, setMode] = useState<'deposit' | 'withdraw'>('deposit');

  const { data: contractBalance, refetch: refetchContract } = useContractBalance(address);
  const { data: usdcBalance, refetch: refetchUSDC } = useUSDCBalance(address);
  const { deposit, isPending: isDepositing } = useDeposit();
  const { withdraw, isPending: isWithdrawing } = useWithdraw();
  const { approve, isPending: isApproving } = useApproveUSDC();
  const { mint, isPending: isMinting } = useMintUSDC();

  if (!isConnected) return null;

  const handleDeposit = async () => {
    await approve(amount);
    await deposit(amount);
    refetchContract();
    refetchUSDC();
  };

  const handleWithdraw = async () => {
    await withdraw(amount);
    refetchContract();
    refetchUSDC();
  };

  const handleMint = async () => {
    if (address) {
      await mint(address, '1000');
      refetchUSDC();
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Wallet Balance</h3>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <div className="text-sm text-gray-500">Wallet USDC</div>
          <div className="text-lg font-medium">
            {usdcBalance ? formatUnits(usdcBalance, 6) : '0'} USDC
          </div>
        </div>
        <div>
          <div className="text-sm text-gray-500">Platform Balance</div>
          <div className="text-lg font-medium">
            {contractBalance ? formatUnits(contractBalance, 6) : '0'} USDC
          </div>
        </div>
      </div>

      <button
        onClick={handleMint}
        disabled={isMinting}
        className="w-full mb-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700 disabled:opacity-50"
      >
        {isMinting ? 'Minting...' : 'Mint 1000 Test USDC'}
      </button>

      <div className="flex gap-2 mb-4">
        <button
          onClick={() => setMode('deposit')}
          className={`flex-1 py-2 rounded-md ${
            mode === 'deposit' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          Deposit
        </button>
        <button
          onClick={() => setMode('withdraw')}
          className={`flex-1 py-2 rounded-md ${
            mode === 'withdraw' ? 'bg-blue-600 text-white' : 'bg-gray-100'
          }`}
        >
          Withdraw
        </button>
      </div>

      <input
        type="number"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        className="w-full border rounded-md p-2 mb-4"
        placeholder="Amount"
      />

      <button
        onClick={mode === 'deposit' ? handleDeposit : handleWithdraw}
        disabled={isDepositing || isWithdrawing || isApproving}
        className="w-full py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
      >
        {isDepositing || isApproving
          ? 'Processing...'
          : isWithdrawing
          ? 'Withdrawing...'
          : mode === 'deposit'
          ? 'Deposit'
          : 'Withdraw'}
      </button>
    </div>
  );
}
