import './style.css'
import { mount } from 'svelte'
import App from './App.svelte'

// Svelte 5: компонент монтируется функцией mount (вместо `new App(...)` из Svelte 3/4)
const app = mount(App, { target: document.getElementById('app') })

export default app
