<script>
  import { onMount } from 'svelte'
  import { LoadSettings, SaveSettings } from '../wailsjs/go/main/App.js'

  let s = $state(null)          // текущие значения формы
  let saved = $state('')        // снимок последних загруженных/сохраненных значений (JSON)
  let message = $state(null)    // { ok, text }
  let saving = $state(false)
  let shown = $state({})        // какие секретные поля сейчас показаны открытым текстом

  const dirty = $derived(s !== null && JSON.stringify(s) !== saved)
  // Файла настроек еще нет — показаны встроенные значения; сохранить можно и без правок
  const canSave = $derived(dirty || (s !== null && !s.configSaved))

  // Описание формы: группа -> поля. path — путь к значению внутри Settings (см. gui/app.go).
  const groups = [
    {
      title: 'Общие',
      fields: [
        { path: 'clientRequestTimeout', label: 'Таймаут запроса', hint: 'Общий лимит на одну проверку, например 10s' },
      ],
    },
    {
      title: 'Apache / 1С',
      fields: [
        { path: 'apache.address', label: 'Адрес', hint: 'Например 192.168.0.75 или localhost' },
        { path: 'apache.port', label: 'Порт', hint: 'Пусто — 80' },
        { path: 'apache.publicationName', label: 'Имя публикации базы' },
        { path: 'apache.confPath', label: 'Путь к httpd.conf', hint: 'Проверяется только для локального Apache' },
        { path: 'apache.warningDuration', label: 'Порог медленного ответа', hint: 'Например 2s' },
      ],
    },
    {
      title: 'DataMobile (пользователь 1С)',
      fields: [
        { path: 'datamobile.user', label: 'Логин' },
        { path: 'datamobile.pass', label: 'Пароль', secret: true },
      ],
    },
    {
      title: 'ЛМ ЧЗ',
      fields: [
        { path: 'lmcz.host', label: 'Хост', hint: 'Пусто — искать сервер сканированием подсети' },
        { path: 'lmcz.subnet', label: 'Подсеть для поиска', hint: 'Первые три октета, например 192.168.0' },
        { path: 'lmcz.port', label: 'Порт', hint: 'Обычно 5998' },
        { path: 'lmcz.scannerTimeout', label: 'Таймаут сканера на хост', hint: 'Например 500ms' },
        { path: 'lmcz.user', label: 'Логин' },
        { path: 'lmcz.pass', label: 'Пароль', secret: true },
        { path: 'lmcz.token', label: 'Токен инициализации', secret: true },
      ],
    },
  ]

  function get(path) {
    return path.split('.').reduce((o, k) => o[k], s)
  }

  function set(path, value) {
    const keys = path.split('.')
    const last = keys.pop()
    keys.reduce((o, k) => o[k], s)[last] = value
  }

  async function load() {
    message = null
    try {
      s = await LoadSettings()
    } catch (e) {
      message = { ok: false, text: String(e) }
      return
    }
    saved = JSON.stringify(s)
  }

  async function save() {
    saving = true
    message = null
    try {
      await SaveSettings($state.snapshot(s))
      s.configSaved = true
      saved = JSON.stringify(s)
      message = { ok: true, text: 'Сохранено' }
    } catch (e) {
      message = { ok: false, text: String(e) }
    } finally {
      saving = false
    }
  }

  onMount(load)
</script>

{#if s}
  <form
    onsubmit={(e) => {
      e.preventDefault()
      save()
    }}
  >
    <p class="path">
      Файл: <code>{s.configPath}</code>
      {#if !s.configSaved}
        <br /><span class="notice">
          Файл еще не создан — используются значения по умолчанию. «Сохранить» создаст его.
        </span>
      {/if}
    </p>

    {#each groups as g}
      <fieldset>
        <legend>{g.title}</legend>
        {#each g.fields as f}
          <label>
            <span class="label">{f.label}</span>
            <span class="input">
              <input
                type={f.secret && !shown[f.path] ? 'password' : 'text'}
                value={get(f.path)}
                oninput={(e) => set(f.path, e.currentTarget.value)}
                spellcheck="false"
                autocomplete="off"
              />
              {#if f.secret}
                <button
                  type="button"
                  class="eye"
                  title={shown[f.path] ? 'Скрыть' : 'Показать'}
                  onclick={() => (shown[f.path] = !shown[f.path])}>{shown[f.path] ? 'Скрыть' : 'Показать'}</button
                >
              {/if}
            </span>
            {#if f.hint}<span class="hint">{f.hint}</span>{/if}
          </label>
        {/each}
      </fieldset>
    {/each}

    <div class="actions">
      <button class="primary" type="submit" disabled={!canSave || saving}>Сохранить</button>
      <button type="button" onclick={load} disabled={saving}>Перечитать файл</button>
      {#if message}
        <span class:ok={message.ok} class:err={!message.ok}>{message.text}</span>
      {:else if dirty}
        <span class="muted">Есть несохраненные изменения</span>
      {/if}
    </div>
  </form>
{:else if message}
  <div class="card err">{message.text}</div>
  <button onclick={load}>Повторить</button>
{/if}

<style>
  form {
    max-width: 760px;
  }

  .path {
    color: var(--muted);
    margin: 0 0 12px;
    user-select: text;
  }

  fieldset {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 12px 16px 4px;
    margin: 0 0 12px;
  }

  legend {
    font-weight: 600;
    padding: 0 6px;
  }

  label {
    display: grid;
    grid-template-columns: 220px 1fr;
    column-gap: 12px;
    align-items: center;
    margin-bottom: 10px;
  }

  .input {
    display: flex;
    gap: 6px;
  }

  .eye {
    flex: none;
    padding: 2px 10px;
    font-size: 12px;
  }

  .hint {
    grid-column: 2;
    color: var(--muted);
    font-size: 12px;
    margin-top: 2px;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 10px;
    position: sticky;
    bottom: 0;
    padding: 10px 0;
    background: var(--bg);
  }

  .notice {
    color: var(--warn);
  }

  .ok {
    color: var(--ok);
  }

  .err {
    color: var(--err);
  }

  .muted {
    color: var(--muted);
  }

  .card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 16px;
    margin-bottom: 12px;
    user-select: text;
  }

  @media (max-width: 760px) {
    label {
      grid-template-columns: 1fr;
    }

    .hint {
      grid-column: 1;
    }
  }
</style>
