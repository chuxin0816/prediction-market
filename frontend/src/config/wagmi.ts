import { connectorsForWallets } from '@rainbow-me/rainbowkit';
import {
  metaMaskWallet,
  coinbaseWallet,
  walletConnectWallet,
  rainbowWallet,
  injectedWallet,
} from '@rainbow-me/rainbowkit/wallets';
import { createConfig, http } from 'wagmi';
import { sepolia } from 'wagmi/chains';

// Get your own project ID from https://cloud.walletconnect.com
const projectId = import.meta.env.VITE_WALLETCONNECT_PROJECT_ID || 'demo-project-id-for-testing';

// Custom wallet list with MetaMask first
const connectors = connectorsForWallets(
  [
    {
      groupName: 'Recommended',
      wallets: [
        metaMaskWallet,
        coinbaseWallet,
        walletConnectWallet,
        rainbowWallet,
      ],
    },
    {
      groupName: 'Other',
      wallets: [
        injectedWallet,
      ],
    },
  ],
  {
    appName: 'Prediction Market',
    projectId,
  }
);

// 使用自定义 RPC 或回退到公共节点
// 推荐在 .env.local 中设置 VITE_RPC_URL（Alchemy/Infura 免费 tier 即可）
// 例如: VITE_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
const rpcUrl = import.meta.env.VITE_RPC_URL as string | undefined;

export const config = createConfig({
  connectors,
  chains: [sepolia],
  transports: {
    [sepolia.id]: http(rpcUrl),
  },
  ssr: false,
});
