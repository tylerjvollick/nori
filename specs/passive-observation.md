# Passive Observation

## Who

- **Shop owners**: Set up cameras/sensors for automated time tracking.
- **Operators**: Benefit from zero-friction time logging — no check-ins needed.
- **The system**: Ingests presence data and creates TimeEvents automatically.

## What

Camera and sensor integration that passively detects which operator is at which
station and automatically logs time. This is the long-term vision for solving
the "it feels unnatural to stop and log" problem — by removing the need to
log at all.

## Where

- Infrastructure: Cameras at each station, connected to a local inference
  service
- Backend: Sensor adapter that receives detection events and creates
  TimeEvents
- Data model: TimeEvents with `source=sensor` (see time-tracking.md)

## Why

Even the lowest-friction manual input (a single tap on a tablet) still requires
the operator to remember and choose to do it. During intense work — complex
joinery, a tricky glue-up — you're not thinking about time tracking. You're
thinking about the joint.

Passive observation removes human action from the loop entirely. The shop
*observes itself* and records what happened. The operator reviews and corrects
after the fact (which is much easier than logging in real time).

This is the most technically ambitious spec and the longest-term investment.
The architecture should be designed now so that sensor data plugs into the
existing TimeEvent system cleanly, but the actual implementation is Phase 2+.

## How

### Near-Term: Tablet Tap-In (Not Truly Passive)

Before cameras, the simplest physical interface:
- A cheap Android tablet mounted at each station
- Displays a simple screen: station name, "Tap to Check In / Check Out"
- NFC tag on the operator's badge for instant identification (tap badge to
  tablet)
- Creates TimeEvent with `source=tap`

This is low-cost, low-complexity, and works today. It bridges the gap between
fully manual and fully passive.

### Medium-Term: Presence Detection

A camera at each station connected to a local inference pipeline:

```
Camera → Frame capture (1 FPS) → Person detection → Station assignment → TimeEvent
```

1. **Camera**: Cheap IP camera (Wyze, Reolink) or USB webcam. One per station.
2. **Frame capture**: 1 frame per second is sufficient — we're detecting
   presence, not actions.
3. **Person detection**: YOLO or similar model running on the local server.
   Detects "a person is present at this station."
4. **Identity** (optional): If using face recognition or badge detection,
   identify *who*. If not, just detect *someone* and let the operator confirm
   identity.
5. **Station assignment**: Camera → station is a fixed mapping (configured
   once during setup).
6. **TimeEvent creation**: When presence is detected for > N seconds
   (debounce), create a `check_in` event. When absence is detected for > N
   seconds, create a `check_out` event.

**Privacy considerations:**
- All processing is local. No frames leave the shop network.
- Frames are processed and discarded — not stored (unless the operator opts
  in for SOP photo capture).
- The system detects *presence*, not specific activities.
- Operators should be informed and consent to camera monitoring.

### Long-Term: Activity Recognition

Beyond presence detection — recognize what the operator is doing:

- Hand planing → tag as "surface preparation"
- Chiseling → tag as "joinery"
- Sanding → tag as "finish prep"
- Spraying → tag as "finishing"

This requires:
- A vision model fine-tuned on woodworking activities
- Significantly more computational resources
- A training dataset (which Nori could help build: operators label a few
  hundred frames, then the model learns)

This is genuinely research-level for domain-specific activities, but the
foundational models (Llama Vision, etc.) are getting good enough that
fine-tuning on a small dataset may be feasible within 1-2 years.

### Architecture: Adapter Pattern

The key design decision: **the sensor system is a plugin, not core**.

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ Camera/Sensor │────→│ Sensor Adapter│────→│ Nori API     │
│ Hardware      │     │ (separate     │     │ (TimeEvent   │
│               │     │  service)     │     │  endpoint)   │
└──────────────┘     └──────────────┘     └──────────────┘
```

- The Sensor Adapter is a separate service (Python, for ML ecosystem access)
- It communicates with Nori via the standard REST API (`POST /time-events`)
- TimeEvents from sensors use `source=sensor` and carry metadata (camera ID,
  confidence score)
- Nori doesn't know or care how the sensor works — it just receives events

This means:
- Nori works fine without any sensors
- Different sensor types can be added independently
- The sensor adapter can be developed and tested separately
- Community contributors can build adapters for their specific hardware

### Frigate Integration (Potential)

[Frigate](https://frigate.video/) is an open-source NVR with real-time
object detection. Many homelab users already run it. A Frigate adapter could:
- Subscribe to Frigate's MQTT events for person detection
- Map Frigate camera names to Nori station IDs
- Create TimeEvents when Frigate detects a person in a station's zone

This would be a compelling integration for the homelab community.

### Confidence and Corrections

Sensor-generated TimeEvents include a confidence score. The UI should:
- Show sensor-logged time entries with a "sensor" badge
- Allow operators to confirm, adjust, or dismiss sensor entries
- Over time, track sensor accuracy and surface it: "Camera-based time
  tracking was 92% accurate this week (compared to manual corrections)"

### API Surface

Sensors use the same TimeEvent API as everything else:

```
POST   /api/spaces/:spaceId/time-events
  {
    "userId": "...",        // from badge/face ID, or null for unknown
    "stationId": "...",     // mapped from camera
    "eventType": "check_in",
    "source": "sensor",
    "timestamp": "...",
    "metadata": {
      "cameraId": "shop-cam-03",
      "confidence": 0.94
    }
  }
```

## Open Questions

- What's the minimum viable sensor setup? (One camera, person detection,
  no identity. Just "someone is at the mill.") This is probably the first
  implementation.
- How should the system handle multiple people at a station? (Especially
  during operations that require two people.)
- Is face recognition acceptable, or is badge/NFC-based identity preferred
  for privacy? (Badge is simpler and less invasive.)
- What frame rate and resolution are needed for reliable person detection?
  (720p at 1 FPS should be sufficient for presence detection.)
- Should the sensor adapter run on the same hardware as Nori, or on a
  dedicated device (e.g., a Raspberry Pi per station with a USB camera)?
