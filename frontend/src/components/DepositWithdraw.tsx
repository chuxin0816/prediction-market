import { useState } from 'react';
import { useAccount } from 'wagmi';
import { formatUnits } from 'viem';
import {
  useContractBalance,
  useUSDCBalance,
  useUSDCAllowance,
  useDeposit,
  useWithdraw,
  useApproveUSDC,
  useMintUSDC,
} from '../hooks/useContract';

export function DepositWithdraw() {
  const { address, isConnected } = useAccount();
  const [amount, setAmount] = useState('100');
  const [mode, setMode] = useState<'deposit' | 'withdraw'>('deposit');
  const [error, setError] = useState<string | null>(null);

  const { data: contractBalance, refetch: refetchContract } = useContractBalance(address);
  const { data: usdcBalance, refetch: refetchUSDC } = useUSDCBalance(address);
  const { data: allowance, refetch: refetchAllowance } = useUSDCAllowance(address);
  const { deposit, isPending: isDepositing } = useDeposit();
  const { withdraw, isPending: isWithdrawing } = useWithdraw();
  const { approve, isPending: isApproving } = useApproveUSDC();
  const { mint, isPending: isMinting } = useMintUSDC();

  if (!isConnected) return null;

  const walletBalance = usdcBalance ? Number(formatUnits(usdcBalance, 6)) : 0;
  const platformBalance = contractBalance ? Number(formatUnits(contractBalance, 6)) : 0;
  const currentAllowance = allowance ? Number(formatUnits(allowance, 6)) : 0;

  const handleDeposit = async () => {
    setError(null);
    const depositAmount = Number(amount);

    // 检查余额是否足够
    if (depositAmount > walletBalance) {
      setError(`USDC 余额不足。你有 $${walletBalance.toFixed(2)}，但想存入 $${depositAmount}`);
      return;
    }

    try {
      // 只有当 allowance 不足时才需要 approve
      if (currentAllowance < depositAmount) {
        await approve(amount);
        // 等待 allowance 更新
        await new Promise(resolve => setTimeout(resolve, 2000));
        refetchAllowance();
      }
      await deposit(amount);
      refetchContract();
      refetchUSDC();
      refetchAllowance();
    } catch (err) {
      setError('交易失败，请重试');
    }
  };

  const handleWithdraw = async () => {
    setError(null);
    const withdrawAmount = Number(amount);

    // 检查平台余额是否足够
    if (withdrawAmount > platformBalance) {
      setError(`平台余额不足。你有 $${platformBalance.toFixed(2)}，但想提取 $${withdrawAmount}`);
      return;
    }

    try {
      await withdraw(amount);
      refetchContract();
      refetchUSDC();
    } catch (err) {
      setError('交易失败，请重试');
    }
  };

  const handleMint = async () => {
    setError(null);
    if (address) {
      try {
        await mint(address, '1000');
        refetchUSDC();
      } catch (err) {
        setError('Mint 失败，请重试');
      }
    }
  };

  const quickAmounts = [50, 100, 500, 1000];

  // 检查当前操作是否需要 approve
  const needsApproval = mode === 'deposit' && currentAllowance < Number(amount);

  return (
    <div className="bg-white rounded-2xl shadow-lg border border-gray-100 overflow-hidden">
      {/* Header */}
      <div className="p-6 border-b border-gray-100">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div>
            <h3 className="font-bold text-gray-900">Fund Management</h3>
            <p className="text-sm text-gray-500">Deposit or withdraw USDC</p>
          </div>
        </div>
      </div>

      {/* Mint Test USDC */}
      <div className="p-6 bg-gradient-to-br from-gray-50 to-white">
        <button
          onClick={handleMint}
          disabled={isMinting}
          className="mt-4 w-full py-2.5 bg-gradient-to-r from-purple-500 to-pink-500 text-white text-sm font-medium rounded-xl hover:from-purple-600 hover:to-pink-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center justify-center gap-2"
        >
          {isMinting ? (
            <>
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              Minting...
            </>
          ) : (
            <>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
              Mint 1000 Test USDC
            </>
          )}
        </button>
      </div>

      {/* Mode Toggle */}
      <div className="px-6 pt-6">
        <div className="flex gap-2 p-1 bg-gray-100 rounded-xl">
          <button
            onClick={() => { setMode('deposit'); setError(null); }}
            className={`flex-1 py-2.5 text-sm font-medium rounded-lg transition-all ${
              mode === 'deposit'
                ? 'bg-white text-gray-900 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <div className="flex items-center justify-center gap-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
              </svg>
              Deposit
            </div>
          </button>
          <button
            onClick={() => { setMode('withdraw'); setError(null); }}
            className={`flex-1 py-2.5 text-sm font-medium rounded-lg transition-all ${
              mode === 'withdraw'
                ? 'bg-white text-gray-900 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            <div className="flex items-center justify-center gap-2">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
              </svg>
              Withdraw
            </div>
          </button>
        </div>
      </div>

      {/* Amount Input */}
      <div className="p-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">Amount (USDC)</label>
        <div className="relative">
          <span className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400 font-medium">$</span>
          <input
            type="number"
            value={amount}
            onChange={(e) => { setAmount(e.target.value); setError(null); }}
            className="w-full pl-8 pr-4 py-3 border border-gray-200 rounded-xl text-lg font-medium focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
            placeholder="0.00"
          />
        </div>

        {/* Quick Amount Buttons */}
        <div className="flex gap-2 mt-3">
          {quickAmounts.map((quickAmount) => (
            <button
              key={quickAmount}
              onClick={() => { setAmount(String(quickAmount)); setError(null); }}
              className={`flex-1 py-2 text-sm font-medium rounded-lg border transition-all ${
                amount === String(quickAmount)
                  ? 'border-blue-500 bg-blue-50 text-blue-600'
                  : 'border-gray-200 text-gray-600 hover:border-gray-300 hover:bg-gray-50'
              }`}
            >
              ${quickAmount}
            </button>
          ))}
        </div>

        {/* Error Message */}
        {error && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-600">{error}</p>
          </div>
        )}

        {/* Action Button */}
        <button
          onClick={mode === 'deposit' ? handleDeposit : handleWithdraw}
          disabled={isDepositing || isWithdrawing || isApproving || !amount || Number(amount) <= 0}
          className={`mt-4 w-full py-3.5 text-white text-sm font-semibold rounded-xl transition-all flex items-center justify-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed ${
            mode === 'deposit'
              ? 'bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700'
              : 'bg-gradient-to-r from-orange-500 to-red-500 hover:from-orange-600 hover:to-red-600'
          }`}
        >
          {isDepositing || isApproving ? (
            <>
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {isApproving ? 'Approving...' : 'Depositing...'}
            </>
          ) : isWithdrawing ? (
            <>
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              Withdrawing...
            </>
          ) : (
            <>
              {mode === 'deposit' ? (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
                </svg>
              )}
              {mode === 'deposit'
                ? (needsApproval ? `Approve & Deposit $${amount || '0'}` : `Deposit $${amount || '0'}`)
                : `Withdraw $${amount || '0'}`}
            </>
          )}
        </button>

        {/* Info Text */}
        <p className="mt-4 text-xs text-center text-gray-400">
          {mode === 'deposit'
            ? 'Deposit USDC from your wallet to start trading'
            : 'Withdraw USDC from platform to your wallet'}
        </p>
      </div>
    </div>
  );
}
