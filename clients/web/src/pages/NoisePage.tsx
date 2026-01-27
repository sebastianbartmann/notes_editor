import { useState, useRef, useCallback } from 'react'
import styles from './NoisePage.module.css'

const BASE_GAIN = 0.24
const LFO_FREQUENCY = 0.07
const LFO_DEPTH = 0.025
const DRIFT_INTERVAL = 2400

interface AudioNodes {
  context: AudioContext
  masterGain: GainNode
  bassSource: AudioBufferSourceNode
  highSource: AudioBufferSourceNode
  lfo: OscillatorNode
  lfoGain: GainNode
}

export default function NoisePage() {
  const [isPlaying, setIsPlaying] = useState(false)
  const nodesRef = useRef<AudioNodes | null>(null)
  const driftTimerRef = useRef<number | null>(null)

  const createNoiseBuffer = (context: AudioContext): AudioBuffer => {
    const bufferSize = context.sampleRate * 2
    const buffer = context.createBuffer(1, bufferSize, context.sampleRate)
    const data = buffer.getChannelData(0)
    for (let i = 0; i < bufferSize; i++) {
      data[i] = Math.random() * 2 - 1
    }
    return buffer
  }

  const createFilteredSource = (
    context: AudioContext,
    buffer: AudioBuffer,
    lowpassFreq: number,
    highpassFreq: number,
    gain: number,
    bassBoostDb: number,
    masterGain: GainNode
  ): AudioBufferSourceNode => {
    const source = context.createBufferSource()
    source.buffer = buffer
    source.loop = true

    // Bass shelf filter
    const bassShelf = context.createBiquadFilter()
    bassShelf.type = 'lowshelf'
    bassShelf.frequency.value = 200
    bassShelf.gain.value = bassBoostDb

    // Lowpass filter
    const lowpass = context.createBiquadFilter()
    lowpass.type = 'lowpass'
    lowpass.frequency.value = lowpassFreq

    // Highpass filter
    const highpass = context.createBiquadFilter()
    highpass.type = 'highpass'
    highpass.frequency.value = highpassFreq

    // Gain node
    const gainNode = context.createGain()
    gainNode.gain.value = gain

    // Connect chain
    source.connect(bassShelf)
    bassShelf.connect(lowpass)
    lowpass.connect(highpass)
    highpass.connect(gainNode)
    gainNode.connect(masterGain)

    return source
  }

  const startDrift = useCallback((masterGain: GainNode) => {
    const drift = () => {
      const driftAmount = (Math.random() - 0.5) * 0.04
      masterGain.gain.setValueAtTime(
        BASE_GAIN + driftAmount,
        masterGain.context.currentTime
      )
    }
    driftTimerRef.current = window.setInterval(drift, DRIFT_INTERVAL)
  }, [])

  const stopDrift = useCallback(() => {
    if (driftTimerRef.current) {
      clearInterval(driftTimerRef.current)
      driftTimerRef.current = null
    }
  }, [])

  const start = useCallback(() => {
    const context = new AudioContext()
    const buffer = createNoiseBuffer(context)

    // Master gain
    const masterGain = context.createGain()
    masterGain.gain.value = BASE_GAIN
    masterGain.connect(context.destination)

    // LFO for subtle volume modulation
    const lfo = context.createOscillator()
    lfo.type = 'sine'
    lfo.frequency.value = LFO_FREQUENCY

    const lfoGain = context.createGain()
    lfoGain.gain.value = LFO_DEPTH

    lfo.connect(lfoGain)
    lfoGain.connect(masterGain.gain)
    lfo.start()

    // Bass layer: 50-900Hz, gain 0.3, +4dB boost
    const bassSource = createFilteredSource(
      context, buffer, 900, 50, 0.3, 4, masterGain
    )

    // High layer: 1200-6000Hz, gain 0.08, 0dB boost
    const highSource = createFilteredSource(
      context, buffer, 6000, 1200, 0.08, 0, masterGain
    )

    bassSource.start()
    highSource.start()

    nodesRef.current = {
      context,
      masterGain,
      bassSource,
      highSource,
      lfo,
      lfoGain,
    }

    startDrift(masterGain)
    setIsPlaying(true)
  }, [startDrift])

  const stop = useCallback(() => {
    stopDrift()
    if (nodesRef.current) {
      const { context, bassSource, highSource, lfo } = nodesRef.current
      bassSource.stop()
      highSource.stop()
      lfo.stop()
      context.close()
      nodesRef.current = null
    }
    setIsPlaying(false)
  }, [stopDrift])

  const toggle = useCallback(() => {
    if (isPlaying) {
      stop()
    } else {
      start()
    }
  }, [isPlaying, start, stop])

  return (
    <div className={styles.page}>
      <h2>Noise Generator</h2>
      <p className={styles.description}>
        Procedural rain-like white noise for focus and relaxation.
      </p>
      <div className={styles.controls}>
        <button
          onClick={toggle}
          className={isPlaying ? 'active' : ''}
        >
          {isPlaying ? 'Stop' : 'Play'}
        </button>
        {isPlaying && (
          <span className={styles.status}>Playing...</span>
        )}
      </div>
    </div>
  )
}
