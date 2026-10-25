<script>
  import { onMount, tick } from 'svelte'
  import { EventsOn } from '../wailsjs/runtime/runtime.js'
  import { CheckDataMobile, LmczStatus, LmczInit } from '../wailsjs/go/main/App.js'
  import Settings from './Settings.svelte'

  const tabs = [
    { id: 'dm', title: 'DataMobile' },
    { id: 'lmcz', title: 'ЛМ ЧЗ' },
    { id: 'settings', title: 'Настройки' },
  ]

  let tab = $state('dm')
  let log = $state([])          // строки лога: { text, kind }
  let running = $state(null)    // ключ выполняемой операции или null
  let results = $state({})      // ключ операции -> { ok, text, time }
  let logEl

  // Определяем цвет строки по ее тексту — сами строки приходят из Go без изменений
  function kindOf(text) {
    if (/ошибк|error|не найден|недоступен|не отвечает|conflict|неверн|failure/i.test(text)) return 'err'
    if (/успех|успешно|just works|в норме|\[\+\]/i.test(text)) return 'ok'
    if (/^-{0,3}\s*$/.test(text)) return 'blank'
    return ''
  }

  async function addLine(text) {
    log.push({ text, kind: kindOf(text) })
    await tick()
    if (logEl) logEl.scrollTop = logEl.scrollHeight
  }

  onMount(() => EventsOn('log', addLine))

  // Запуск операции на стороне Go. Ход работы приходит событиями "log",
  // итог (успех/ошибка) — результатом промиса.
  async function exec(key, call, describe) {
    if (running) return
    running = key
    if (log.length) addLine('')
    addLine(`──── ${new Date().toLocaleTimeString()} ────`)
    try {
      const value = await call()
      results[key] = { ok: true, text: describe(value), time: new Date() }
    } catch (e) {
      results[key] = { ok: false, text: String(e), time: new Date() }
    } finally {
      running = null
    }
  }

  const runDm = () => exec('dm', CheckDataMobile, () => 'Проверка пройдена')
  const runStatus = () =>
    exec('status', LmczStatus, (r) => `(${r.httpCode}) status: ${r.status || '—'}`)
  const runInit = () => exec('init', LmczInit, () => 'Готово')

  function copyLog() {
    navigator.clipboard?.writeText(log.map((l) => l.text).join('\n'))
  }
</script>

{#snippet result(key)}
  {#if running === key}
    <span class="badge busy">Выполняется…</span>
  {:else if results[key]}
    <span class="badge" class:ok={results[key].ok} class:err={!results[key].ok}>
      {results[key].ok ? '✓' : '✗'} {results[key].text}
    </span>
    <span class="time">{results[key].time.toLocaleTimeString()}</span>
  {/if}
{/snippet}

<div class="layout">
  <nav>
    <span class="brand">golm</span>
    {#each tabs as t}
      <button class="tab" class:active={tab === t.id} onclick={() => (tab = t.id)}>{t.title}</button>
    {/each}
  </nav>

  {#if tab === 'settings'}
    <main class="scroll">
      <Settings />
    </main>
  {:else}
    <main>
      {#if tab === 'dm'}
        <section class="card">
          <h2>Диагностика DataMobile – Apache – 1С</h2>
          <p class="muted">
            Проверяет, что Apache отвечает, в httpd.conf подключен модуль 1С (для локального Apache)
            и HTTP-сервис DataMobileExch опубликованной базы доступен и отвечает быстро.
          </p>
          <div class="row">
            <button class="primary" disabled={!!running} onclick={runDm}>Проверить</button>
            {@render result('dm')}
          </div>
        </section>
      {:else}
        <section class="card">
          <h2>Локальный модуль «Честного знака»</h2>
          <p class="muted">
            Если адрес сервера в настройках не задан, он ищется сканированием подсети.
          </p>
          <div class="row">
            <button class="primary" disabled={!!running} onclick={runStatus}>Проверить статус</button>
            {@render result('status')}
          </div>
          <div class="row">
            <button disabled={!!running} onclick={runInit}>Инициализировать</button>
            {@render result('init')}
          </div>
          <p class="muted small">
            Инициализация отправляет токен из настроек, только если сервер в состоянии
            <code>not_configured</code>.
          </p>
        </section>
      {/if}

      <section class="log">
        <header>
          <span>Лог</span>
          <span class="spacer"></span>
          <button class="small-btn" onclick={copyLog} disabled={!log.length}>Копировать</button>
          <button class="small-btn" onclick={() => (log = [])} disabled={!log.length || !!running}>Очистить</button>
        </header>
        <pre bind:this={logEl}>{#each log as line}<div class={line.kind}>{line.text || ' '}</div>{:else}<div class="placeholder">Здесь появится ход проверки</div>{/each}</pre>
      </section>
    </main>
  {/if}
</div>

<style>
  .layout {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  nav {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 8px 16px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
  }

  .brand {
    font-weight: 700;
    margin-right: 16px;
    letter-spacing: 0.5px;
  }

  .tab {
    border-color: transparent;
    background: transparent;
  }

  .tab.active {
    background: var(--bg);
    border-color: var(--border);
    font-weight: 600;
  }

  main {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
  }

  main.scroll {
    overflow-y: auto;
  }

  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px;
  }

  h2 {
    margin: 0 0 6px;
    font-size: 16px;
  }

  .muted {
    color: var(--muted);
    margin: 0 0 12px;
  }

  .small {
    font-size: 12px;
    margin: 8px 0 0;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .row + .row {
    margin-top: 10px;
  }

  .row button {
    min-width: 160px;
  }

  .badge {
    padding: 3px 10px;
    border-radius: 999px;
    font-size: 13px;
    background: var(--bg);
    user-select: text;
  }

  .badge.ok {
    color: var(--ok);
    background: var(--ok-bg);
  }

  .badge.err {
    color: var(--err);
    background: var(--err-bg);
  }

  .badge.busy {
    color: var(--muted);
  }

  .time {
    color: var(--muted);
    font-size: 12px;
  }

  .log {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    border-radius: var(--radius);
    overflow: hidden;
    border: 1px solid var(--border);
  }

  .log header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 10px;
    background: var(--surface);
    border-bottom: 1px solid var(--border);
    font-weight: 600;
  }

  .spacer {
    flex: 1;
  }

  .small-btn {
    padding: 2px 10px;
    font-size: 12px;
    font-weight: 400;
  }

  pre {
    flex: 1;
    margin: 0;
    padding: 10px 12px;
    overflow: auto;
    background: var(--log-bg);
    color: var(--log-text);
    font-family: Consolas, "Cascadia Mono", monospace;
    font-size: 13px;
    line-height: 1.45;
    user-select: text;
  }

  pre .ok {
    color: #6fd39a;
  }

  pre .err {
    color: #ff8a80;
  }

  pre .placeholder {
    color: #6b7686;
  }
</style>
