import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { writeFileSync } from 'node:fs'

// vite build очищает dist/ перед сборкой и удаляет dist/.gitkeep.
// Возвращаем его: папка должна быть в git, иначе //go:embed в main.go
// не скомпилируется (go vet / go test) без предварительной сборки фронтенда.
const keepDist = {
  name: 'keep-dist',
  closeBundle() {
    writeFileSync('dist/.gitkeep', '')
  },
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte(), keepDist],
})
