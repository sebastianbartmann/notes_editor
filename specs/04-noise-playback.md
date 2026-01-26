# Noise Playback

## Purpose

Noise Playback provides ambient rain sound generation for focus, relaxation, or sleep. The feature is implemented differently on each platform to leverage native capabilities: the web version uses the Web Audio API for synthesized audio, while Android uses a foreground service with pre-recorded audio and system media controls.

## Web Implementation

### Architecture

The web implementation is entirely client-side, using the Web Audio API to generate rain-like white noise in real time. No server-side audio processing or files are involved.

### Audio Generation

The rain sound is created by layering filtered white noise:

1. **Noise Buffer Creation** - A 2-second buffer of random samples (-1 to 1) is generated and looped continuously

2. **Rain Layers** - Two noise layers create the rain texture:
   - **Low layer** (bass/body): Lowpass 900Hz, highpass 50Hz, gain 0.3, bass boost +4dB at 180Hz
   - **High layer** (texture/detail): Lowpass 6000Hz, highpass 1200Hz, gain 0.08, no bass boost

3. **Signal Chain** (per layer):
   ```
   BufferSource -> LowShelf -> Lowpass -> Highpass -> Gain -> MasterGain -> Destination
   ```

### Natural Variation

To prevent the sound from feeling static:

- **LFO Modulation** - A 0.07Hz sine wave oscillator modulates the master gain by +/-0.025
- **Drift Timer** - Every 2.4 seconds, a random drift of +/-0.02 is added to the base gain (0.24)

### Controls

| Control | Action |
|---------|--------|
| Play/Pause | Toggle between playing and paused states |
| Stop | Fully stop playback and reset audio context |

### Status Display

Three states shown in the UI:
- **Stopped** - Initial state, no audio resources allocated
- **Playing** - Audio actively playing
- **Paused** - Audio paused, resources retained

### Browser Compatibility

Falls back to `webkitAudioContext` for older Safari versions. Disables controls if Web Audio API is unavailable.

## Android Implementation

### Architecture

Android uses a foreground service (`NoiseService`) with `MediaPlayer` to play a pre-recorded audio file (`res/raw/noise.mp3`). This approach:

- Allows playback to continue when the app is backgrounded or screen is off
- Integrates with Android's media system for lock screen and notification controls
- Provides system-wide pause/play via Bluetooth headsets and media buttons

### NoiseService

A `Service` that manages audio playback and exposes media controls:

**Intent Actions:**
- `ACTION_PLAY` - Start or resume playback
- `ACTION_PAUSE` - Pause playback
- `ACTION_STOP` - Stop playback and terminate service
- `ACTION_TOGGLE` - Toggle between play/pause

**Lifecycle:**
- `onCreate()` - Creates notification channel and initializes MediaSession
- `onStartCommand()` - Handles intent actions
- `onDestroy()` - Releases MediaPlayer resources

### Seamless Looping

To avoid gaps when the audio file loops, the service uses dual MediaPlayers:

```kotlin
currentPlayer?.setNextMediaPlayer(nextPlayer)
```

When the current player completes:
1. The next player becomes current and continues playing
2. A new next player is prepared
3. The completed player is released

### MediaSession Integration

`MediaSessionCompat` provides:
- Lock screen media controls
- Bluetooth/headset button handling
- Playback state reporting to the system

Supported actions:
- `ACTION_PLAY`
- `ACTION_PAUSE`
- `ACTION_PLAY_PAUSE`
- `ACTION_STOP`

### Notification

A media-style notification with:
- **Title:** "White Noise"
- **Status:** "Playing" or "Paused"
- **Actions:** Play/Pause toggle, Stop
- **Channel:** "Noise Playback" (low importance, no sound)
- **Visibility:** Public (shows on lock screen)
- **Tap action:** Opens MainActivity

The notification is ongoing while playing and auto-dismisses on stop via `stopForeground(STOP_FOREGROUND_REMOVE)`.

### UI Screen (ToolNoiseScreen)

Jetpack Compose screen with:
- Status display ("Playing" / "Stopped")
- Large Play/Pause button
- Large Stop button

The screen sends intents to `NoiseService` via `ContextCompat.startForegroundService()`.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/tools/noise` | Serves the noise playback page (web only) |

The endpoint requires person selection and applies the user's theme. No backend audio processing occurs.

## Platform Differences

| Aspect | Web | Android |
|--------|-----|---------|
| Audio source | Synthesized (Web Audio API) | Pre-recorded file (noise.mp3) |
| Playback location | Browser tab | Foreground service |
| Background playback | Only while tab is active | Continues in background |
| System integration | None | MediaSession, lock screen controls |
| Notification | None | Media-style with play/pause/stop |
| Looping | Native buffer looping | Dual MediaPlayer handoff |
| Sound character | Layered filtered noise | Recorded audio file |

## Key Files

| File | Purpose |
|------|---------|
| `server/web_app/templates/noise.html` | Web UI and Web Audio implementation |
| `server/web_app/main.py` | Route handler for `/tools/noise` |
| `app/android/.../ToolNoiseScreen.kt` | Android Compose UI |
| `app/android/.../NoiseService.kt` | Android foreground service |
| `app/android/app/src/main/res/raw/noise.mp3` | Android audio file |
