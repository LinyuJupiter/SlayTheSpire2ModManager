import {createApp} from 'vue'
import App from './App.vue'
import './style.css'

// 阻止 Ctrl+滚轮 / 触控板缩放页面（与 WebView2 选项配合）
function preventBrowserZoom() {
  const block = (e) => {
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault()
    }
  }
  window.addEventListener('wheel', block, {passive: false})
  window.addEventListener('keydown', (e) => {
    if (!(e.ctrlKey || e.metaKey)) return
    const k = e.key
    if (k === '+' || k === '-' || k === '=' || k === '0') {
      e.preventDefault()
    }
  })
}

preventBrowserZoom()
createApp(App).mount('#app')
