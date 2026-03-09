<script setup>
import { theme } from 'ant-design-vue'
</script>

<template>
  <a-config-provider
    :theme="{
      algorithm: theme.darkAlgorithm,
      token: {
        colorPrimary: '#89b4fa',
        colorBgContainer: '#252536',
        colorBgElevated: '#252536',
        colorBgLayout: '#1e1e2e',
        colorBgSpotlight: '#313244',
        colorBorder: '#313244',
        colorBorderSecondary: '#313244',
        colorText: '#cdd6f4',
        colorTextSecondary: '#a6adc8',
        colorTextTertiary: '#6c7086',
        colorTextQuaternary: '#6c7086',
        colorFill: '#313244',
        colorFillSecondary: '#45475a',
        colorFillTertiary: '#313244',
        colorFillQuaternary: '#252536',
        colorSuccess: '#a6e3a1',
        colorWarning: '#f9e2af',
        colorError: '#f38ba8',
        colorInfo: '#89b4fa',
        borderRadius: 6,
        fontFamily: `'Segoe UI', -apple-system, BlinkMacSystemFont, sans-serif`,
        fontSize: 14,
        colorBgMask: 'rgba(0, 0, 0, 0.6)',
      },
      components: {
        Tree: {
          directoryNodeSelectedBg: '#45475a',
          nodeSelectedBg: '#45475a',
          nodeHoverBg: '#313244',
        },
        Menu: {
          itemBg: '#252536',
          itemHoverBg: '#313244',
          itemSelectedBg: '#45475a',
          darkItemBg: '#252536',
        },
        Modal: {
          contentBg: '#252536',
          headerBg: '#252536',
          footerBg: '#252536',
        },
        Button: {
          defaultBg: '#313244',
          defaultBorderColor: '#313244',
        },
        Input: {
          activeBg: '#1e1e2e',
          hoverBg: '#1e1e2e',
          colorBgContainer: '#1e1e2e',
        },
        Select: {
          colorBgContainer: '#1e1e2e',
          optionActiveBg: '#313244',
          optionSelectedBg: '#45475a',
        },
      }
    }"
  >
    <div id="app-root">
      <slot />
    </div>
  </a-config-provider>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app-root, #__nuxt {
  height: 100%;
  width: 100%;
  overflow: hidden;
}

:root {
  --sidebar-width: 280px;
  --header-height: 40px;
  --bg-primary: #1e1e2e;
  --bg-secondary: #181825;
  --bg-surface: #252536;
  --bg-hover: #313244;
  --bg-active: #45475a;
  --text-primary: #cdd6f4;
  --text-secondary: #a6adc8;
  --text-muted: #6c7086;
  --border-color: #313244;
  --accent: #89b4fa;
  --accent-hover: #74c7ec;
  --danger: #f38ba8;
  --success: #a6e3a1;
  --warning: #f9e2af;
}

body {
  font-family: 'Segoe UI', -apple-system, BlinkMacSystemFont, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 14px;
  user-select: none;
  -webkit-user-select: none;
}

::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--bg-active);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

/* Ant Design 全局暗色覆盖 */
.ant-modal-mask {
  background: rgba(0, 0, 0, 0.6) !important;
}

.ant-btn-text {
  color: var(--text-secondary) !important;
}

.ant-btn-text:hover {
  color: var(--text-primary) !important;
  background: var(--bg-hover) !important;
}

.ant-segmented {
  background: var(--bg-primary) !important;
  color: var(--text-secondary) !important;
}

.ant-segmented-item-selected {
  background: var(--bg-active) !important;
  color: var(--text-primary) !important;
}
</style>
