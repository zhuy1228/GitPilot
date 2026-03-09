export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  ssr: false,
  nitro: {
    preset: 'static',
    output: {
      publicDir: './dist'
    }
  },
  vite: {
    optimizeDeps: {
      exclude: ['@wailsio/runtime']
    }
  },
  plugins: [
    '~/plugins/antd',
  ],
  transpile: ['ant-design-vue', '@ant-design/icons-vue']
})
