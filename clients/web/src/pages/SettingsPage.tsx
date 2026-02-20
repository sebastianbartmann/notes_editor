import { useState, useEffect } from 'react'
import { usePerson } from '../hooks/usePerson'
import { useTheme } from '../hooks/useTheme'
import { useAuth } from '../hooks/useAuth'
import { downloadApk, downloadVaultBackup, fetchEnv, saveEnv } from '../api/settings'
import { getAgentConfig, getAgentGatewayHealth, saveAgentConfig } from '../api/agent'
import { MAX_TEXT_SCALE, MIN_TEXT_SCALE, TEXT_SCALE_STEP } from '../context/ThemeContext'
import styles from './SettingsPage.module.css'

const VERBOSE_OUTPUT_KEY = 'notes_agent_verbose_output'
const LEGACY_SHOW_TOOL_CALLS_KEY = 'notes_agent_show_tool_calls'

export default function SettingsPage() {
  const { person, setPerson } = usePerson()
  const { theme, setTheme, textScale, stepTextScale, resetTextScale } = useTheme()
  const { logout } = useAuth()
  const [envContent, setEnvContent] = useState('')
  const [envStatus, setEnvStatus] = useState('')
  const [isSavingEnv, setIsSavingEnv] = useState(false)
  const [runtimeMode, setRuntimeMode] = useState('gateway_subscription')
  const [agentPrompt, setAgentPrompt] = useState('')
  const [agentStatus, setAgentStatus] = useState('')
  const [gatewayStatus, setGatewayStatus] = useState('Checking gateway...')
  const [isSavingAgent, setIsSavingAgent] = useState(false)
  const [backupStatus, setBackupStatus] = useState('')
  const [isDownloadingBackup, setIsDownloadingBackup] = useState(false)
  const [apkStatus, setApkStatus] = useState('')
  const [isDownloadingApk, setIsDownloadingApk] = useState(false)
  const [verboseOutput, setVerboseOutput] = useState<boolean>(() => {
    const stored = localStorage.getItem(VERBOSE_OUTPUT_KEY)
    if (stored !== null) return stored !== 'false'
    const legacy = localStorage.getItem(LEGACY_SHOW_TOOL_CALLS_KEY)
    return legacy !== 'false'
  })

  useEffect(() => {
    localStorage.setItem(VERBOSE_OUTPUT_KEY, verboseOutput ? 'true' : 'false')
    localStorage.removeItem(LEGACY_SHOW_TOOL_CALLS_KEY)
  }, [verboseOutput])

  useEffect(() => {
    fetchEnv()
      .then(data => setEnvContent(data.content))
      .catch(err => setEnvStatus(`Failed to load .env: ${err instanceof Error ? err.message : err}`))
  }, [])

  useEffect(() => {
    if (!person) return
    getAgentConfig()
      .then(cfg => {
        setRuntimeMode(cfg.runtime_mode)
        setAgentPrompt(cfg.prompt)
        setAgentStatus('')
      })
      .catch(err => setAgentStatus(`Failed to load agent config: ${err instanceof Error ? err.message : err}`))

    getAgentGatewayHealth()
      .then(health => {
        if (!health.configured) {
          setGatewayStatus('Gateway: not configured')
          return
        }
        if (!health.reachable) {
          setGatewayStatus(`Gateway: unreachable (${health.last_error ?? 'connection failed'})`)
          return
        }
        if (!health.healthy) {
          setGatewayStatus(`Gateway: unhealthy (${health.last_error ?? 'reported unhealthy'})`)
          return
        }
        setGatewayStatus(`Gateway: healthy (${health.mode ?? 'unknown mode'})`)
      })
      .catch(err => {
        setGatewayStatus(`Gateway: check failed (${err instanceof Error ? err.message : err})`)
      })
  }, [person])

  const handleSaveEnv = async () => {
    if (isSavingEnv) return
    setIsSavingEnv(true)
    setEnvStatus('')
    try {
      await saveEnv({ content: envContent })
      setEnvStatus('Saved .env')
    } catch (err) {
      setEnvStatus(`Save failed: ${err instanceof Error ? err.message : err}`)
    } finally {
      setIsSavingEnv(false)
    }
  }

  const handleSaveAgent = async () => {
    if (isSavingAgent || !person) return
    setIsSavingAgent(true)
    setAgentStatus('')
    try {
      await saveAgentConfig({
        runtime_mode: runtimeMode,
        prompt: agentPrompt,
      })
      setAgentStatus('Saved agent settings')
    } catch (err) {
      setAgentStatus(`Save failed: ${err instanceof Error ? err.message : err}`)
    } finally {
      setIsSavingAgent(false)
    }
  }

  const handleDownloadBackup = async () => {
    if (isDownloadingBackup || !person) return
    setIsDownloadingBackup(true)
    setBackupStatus('')
    try {
      await downloadVaultBackup()
      setBackupStatus('Backup download started')
    } catch (err) {
      setBackupStatus(`Backup failed: ${err instanceof Error ? err.message : err}`)
    } finally {
      setIsDownloadingBackup(false)
    }
  }

  const handleDownloadApk = async () => {
    if (isDownloadingApk) return
    setIsDownloadingApk(true)
    setApkStatus('')
    try {
      await downloadApk()
      setApkStatus('APK download started')
    } catch (err) {
      setApkStatus(`APK download failed: ${err instanceof Error ? err.message : err}`)
    } finally {
      setIsDownloadingApk(false)
    }
  }

  return (
    <div className={styles.page}>
      <h2>Settings</h2>

      <section className={styles.section}>
        <h3>Person</h3>
        <div className={styles.options}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="person"
              value="sebastian"
              checked={person === 'sebastian'}
              onChange={() => setPerson('sebastian')}
            />
            Sebastian
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="person"
              value="petra"
              checked={person === 'petra'}
              onChange={() => setPerson('petra')}
            />
            Petra
          </label>
        </div>
        {!person && (
          <p className={styles.hint}>
            Please select a person to access your notes.
          </p>
        )}
      </section>

      <section className={styles.section}>
        <h3>Theme</h3>
        <div className={styles.options}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="theme"
              value="dark"
              checked={theme === 'dark'}
              onChange={() => setTheme('dark')}
            />
            Dark
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="theme"
              value="light"
              checked={theme === 'light'}
              onChange={() => setTheme('light')}
            />
            Light
          </label>
        </div>
      </section>

      <section className={styles.section}>
        <h3>Text size</h3>
        <p className={styles.hint}>Global reading/editing size for notes, files, and agent chat.</p>
        <div className={styles.textScaleActions}>
          <button onClick={() => stepTextScale(-1)} disabled={textScale <= MIN_TEXT_SCALE}>
            A-
          </button>
          <button onClick={resetTextScale}>
            Reset
          </button>
          <button onClick={() => stepTextScale(1)} disabled={textScale >= MAX_TEXT_SCALE}>
            A+
          </button>
        </div>
        <p className={styles.hint}>Current: {Math.round(textScale * 100)}% (step {Math.round(TEXT_SCALE_STEP * 100)}%)</p>
      </section>

      <section className={styles.section}>
        <h3>Agent</h3>
        <p className={styles.hint}>Per-person runtime mode and system prompt (`agents.md`).</p>
        <div className={styles.options}>
          <label className={styles.radio}>
            <input
              type="radio"
              name="runtime_mode"
              value="anthropic_api_key"
              checked={runtimeMode === 'anthropic_api_key'}
              onChange={() => setRuntimeMode('anthropic_api_key')}
            />
            Anthropic API Key
          </label>
          <label className={styles.radio}>
            <input
              type="radio"
              name="runtime_mode"
              value="gateway_subscription"
              checked={runtimeMode === 'gateway_subscription'}
              onChange={() => setRuntimeMode('gateway_subscription')}
            />
            Gateway Subscription (Pi)
          </label>
        </div>
        <label className={styles.checkboxRow}>
          <input
            type="checkbox"
            checked={verboseOutput}
            onChange={e => setVerboseOutput(e.target.checked)}
          />
          Enable verbose output (tool calls, gateway status, usage)
        </label>
        <textarea
          value={agentPrompt}
          onChange={e => setAgentPrompt(e.target.value)}
          className={styles.envEditor}
          rows={10}
          spellCheck={false}
        />
        <div className={styles.envActions}>
          <button onClick={handleSaveAgent} disabled={isSavingAgent || !person}>
            {isSavingAgent ? 'Saving...' : 'Save Agent'}
          </button>
          {agentStatus && <span className={styles.envStatus}>{agentStatus}</span>}
        </div>
        <p className={styles.hint}>{gatewayStatus}</p>
      </section>

      <section className={styles.section}>
        <h3>Backup</h3>
        <p className={styles.hint}>Download a compressed copy of the selected person's vault.</p>
        <div className={styles.envActions}>
          <button onClick={handleDownloadBackup} disabled={isDownloadingBackup || !person}>
            {isDownloadingBackup ? 'Preparing backup...' : 'Download backup (.zip)'}
          </button>
          {backupStatus && <span className={styles.envStatus}>{backupStatus}</span>}
        </div>
      </section>

      <section className={styles.section}>
        <h3>Android APK</h3>
        <p className={styles.hint}>Download the latest debug APK built on the server.</p>
        <div className={styles.envActions}>
          <button onClick={handleDownloadApk} disabled={isDownloadingApk}>
            {isDownloadingApk ? 'Preparing APK...' : 'Download APK'}
          </button>
          {apkStatus && <span className={styles.envStatus}>{apkStatus}</span>}
        </div>
      </section>

      <section className={styles.section}>
        <h3>Server .env</h3>
        <p className={styles.hint}>Edit environment variables stored on the server.</p>
        <textarea
          value={envContent}
          onChange={e => setEnvContent(e.target.value)}
          className={styles.envEditor}
          rows={8}
          spellCheck={false}
        />
        <div className={styles.envActions}>
          <button onClick={handleSaveEnv} disabled={isSavingEnv}>
            {isSavingEnv ? 'Saving...' : 'Save'}
          </button>
          {envStatus && <span className={styles.envStatus}>{envStatus}</span>}
        </div>
      </section>

      <section className={styles.section}>
        <h3>Account</h3>
        <button onClick={logout} className="danger">
          Logout
        </button>
      </section>
    </div>
  )
}
