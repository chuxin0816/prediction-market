import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [react()],
    define: {
      'import.meta.env.VITE_API_URL': JSON.stringify(env.VITE_API_URL || 'http://10.251.229.114:8080/api'),
      'import.meta.env.VITE_CONTRACT_ADDRESS': JSON.stringify(env.VITE_CONTRACT_ADDRESS || '0x614Ec82A607e6604ba7c2A431ec271986ed2367d'),
      'import.meta.env.VITE_USDC_ADDRESS': JSON.stringify(env.VITE_USDC_ADDRESS || '0x9B419c8A1002b4D98098326d8bF32a038133D11e'),
    },
  }
})
