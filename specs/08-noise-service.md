# Noise Service Specification

> Status: Draft
> Version: 1.0
> Last Updated: 2026-01-18

## Overview

The Noise Service provides white noise playback for sleep and focus purposes. Two separate implementations exist:

1. **Android** - Foreground service with MediaPlayer and MediaSession integration
2. **Web** - Browser-based using Web Audio API with procedural noise generation

---

## Android Implementation

### NoiseService.kt

A foreground service that plays white noise audio with system media controls.

**Package:** `com.bartmann.noteseditor`

**Class:** `NoiseService : Service()`

#### Properties

```kotlin
private var currentPlayer: MediaPlayer? = null
private var nextPlayer: MediaPlayer? = null
private var mediaSession: MediaSessionCompat? = null
private var isPlaying = false
```

#### Constants

```kotlin
const val CHANNEL_ID = "noise_playback"
const val NOTIFICATION_ID = 2001
const val ACTION_TOGGLE = "com.bartmann.noteseditor.noise.TOGGLE"
const val ACTION_PLAY = "com.bartmann.noteseditor.noise.PLAY"
const val ACTION_PAUSE = "com.bartmann.noteseditor.noise.PAUSE"
const val ACTION_STOP = "com.bartmann.noteseditor.noise.STOP"
```

#### Service Lifecycle

| Method | Behavior |
|--------|----------|
| `onCreate()` | Creates notification channel, initializes MediaSession |
| `onStartCommand()` | Handles actions (PLAY/PAUSE/STOP/TOGGLE) |
| `onDestroy()` | Releases players |
| `onBind()` | Returns null (not a bound service) |

#### Seamless Looping

Uses two MediaPlayer instances for gapless playback:

```kotlin
private fun ensurePlayer() {
    if (currentPlayer == null) {
        currentPlayer = createPlayer().also { attachCompletionListener(it) }
    }
    if (nextPlayer == null) {
        nextPlayer = createPlayer()
    }
    currentPlayer?.setNextMediaPlayer(nextPlayer)
}

private fun createPlayer(): MediaPlayer = MediaPlayer.create(this, R.raw.noise)
```

Completion listener rotation:
1. When current finishes, nextPlayer becomes current
2. New nextPlayer is created
3. `setNextMediaPlayer` chains them
4. Old player is released

#### Actions

| Action | Behavior |
|--------|----------|
| ACTION_PLAY | Start playback, show notification |
| ACTION_PAUSE | Pause playback, update notification |
| ACTION_STOP | Stop playback, remove notification, stop service |
| ACTION_TOGGLE | Toggle between play/pause |

#### Notification

- Channel: "Noise Playback" with IMPORTANCE_LOW
- MediaStyle notification with MediaSession token
- Actions: Play/Pause toggle, Stop button
- Compact view shows both actions
- Ongoing notification while playing
- Delete intent triggers stop

#### MediaSession

- Callbacks: onPlay, onPause, onStop
- PlaybackState updated with current state
- Supported actions: PLAY, PAUSE, PLAY_PAUSE, STOP

### NoisePlaybackState.kt

Singleton state object for sharing playback state between NoiseService and UI.

**Package:** `com.bartmann.noteseditor`

**Object:** `NoisePlaybackState`

```kotlin
object NoisePlaybackState {
    var isPlaying by mutableStateOf(false)
        internal set  // Only NoiseService can modify
}
```

**Purpose:**
- Bridges NoiseService state to Compose UI reactively
- `internal set` ensures only the service updates the state
- UI observes `isPlaying` to reflect current playback status

### ToolNoiseScreen.kt

Compose UI for controlling the noise service.

**Composable:** `ToolNoiseScreen(modifier, padding)`

#### State

```kotlin
// Observes NoisePlaybackState.isPlaying for reactive updates
```

#### UI Elements

- Status display: "Playing" or "Stopped"
- Play/Pause button (toggles state)
- Stop button

#### Service Control

```kotlin
private fun toggleNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
        .setAction(NoiseService.ACTION_TOGGLE)
    ContextCompat.startForegroundService(context, intent)
}

private fun stopNoise(context: Context) {
    val intent = Intent(context, NoiseService::class.java)
        .setAction(NoiseService.ACTION_STOP)
    context.startService(intent)
}
```

### Manifest Declaration

```xml
<service
    android:name=".NoiseService"
    android:exported="false"
    android:foregroundServiceType="mediaPlayback" />
```

---

## Web Implementation

### noise.html

Browser-based white noise using Web Audio API.

**Route:** `GET /tools/noise`

#### Audio Context Setup

```javascript
audioCtx = new (window.AudioContext || window.webkitAudioContext)();
masterGain = audioCtx.createGain();
masterGain.gain.value = 0.24;  // baseGain
masterGain.connect(audioCtx.destination);
```

#### Noise Generation

Creates white noise buffer:

```javascript
function createNoiseBuffer() {
    const bufferSize = 2 * audioCtx.sampleRate;
    const buffer = audioCtx.createBuffer(1, bufferSize, audioCtx.sampleRate);
    const data = buffer.getChannelData(0);
    for (let i = 0; i < bufferSize; i++) {
        data[i] = Math.random() * 2 - 1;
    }
    return buffer;
}
```

#### Rain Sound Layers

Two filtered noise layers create rain-like texture:

| Layer | Lowpass | Highpass | Gain | Bass Boost |
|-------|---------|----------|------|------------|
| Bass layer | 900 Hz | 50 Hz | 0.3 | +4 dB |
| High layer | 6000 Hz | 1200 Hz | 0.08 | 0 dB |

Filter chain per layer:
```
NoiseSource → LowShelf → Lowpass → Highpass → Gain → MasterGain
```

#### LFO Modulation

Subtle volume variation for natural feel:

```javascript
lfo = audioCtx.createOscillator();
lfo.type = 'sine';
lfo.frequency.value = 0.07;  // Very slow modulation
lfoGain = audioCtx.createGain();
lfoGain.gain.value = 0.025;  // Subtle depth
lfo.connect(lfoGain);
lfoGain.connect(masterGain.gain);
```

#### Drift Timer

Additional random gain drift every 2.4 seconds:

```javascript
driftTimer = setInterval(() => {
    const drift = (Math.random() - 0.5) * 0.04;
    masterGain.gain.value = baseGain + drift;
}, 2400);
```

#### States

| State | Status Text | Toggle Button |
|-------|-------------|---------------|
| stopped | "Stopped" | "Play" |
| playing | "Playing" | "Pause" |
| paused | "Paused" | "Play" |

#### Browser Compatibility

Checks for AudioContext support, disables controls if unavailable.

---

## User Interface

### Android

- Dedicated screen accessible from Tools navigation
- Simple status indicator (Playing/Stopped)
- Two buttons: Play/Pause toggle and Stop
- System media controls via notification
- Lock screen and notification shade controls

### Web

- Accessible at `/tools/noise`
- Status display with three states
- Two buttons: Toggle (Play/Pause) and Stop
- Requires tab to remain open for playback

---

## Comparison

| Feature | Android | Web |
|---------|---------|-----|
| Audio Source | Pre-recorded file (R.raw.noise) | Procedural generation |
| Looping | MediaPlayer chaining | BufferSource.loop = true |
| Background | Foreground service | Tab must stay open |
| System Integration | MediaSession, notification controls | None |
| Sound Character | Fixed audio file | Rain-like with modulation |

---

## Integration Notes

### Android

- Service requires `FOREGROUND_SERVICE` permission
- Notification channel created on service startup
- MediaSession enables external control (Bluetooth, Android Auto)
- Player rotation ensures seamless looping without gaps

### Web

- Uses WebKit prefix fallback for older browsers
- AudioContext may require user gesture to start (browser policy)
- LFO and drift provide organic variation to procedural noise
- All audio nodes disconnected on stop to prevent memory leaks
