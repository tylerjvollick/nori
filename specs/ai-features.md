# AI Features

## Who

- **Operators**: Receive contextual prompts during work, voice/photo input.
- **Managers / owners**: Get AI-generated summaries, recipe improvement
  suggestions, plain-language bottleneck reports.
- **The system**: Uses AI for auto-tagging, transcription, and content
  structuring.

## What

Embedded AI capabilities powered by a local LLM or cloud provider (BYOK).
These are features *within* Nori itself — not the CLI skill that exposes Nori
to external AI agents (see cli.md). The embedded AI uses the MCP server
(see mcp-server.md) as its tool protocol to interact with Nori's service
layer.

The AI layer handles: recipe refinement suggestions, first-time capture
assistance, voice-to-text transcription, photo annotation, bottleneck
summaries in plain language, and auto-tagging.

### Two AI Integration Layers

1. **External AI agents** (OpenCode, Claude Code, Open Claw): Use the CLI
   via a skill file. The AI runs outside Nori and operates it through shell
   commands. This is the v1 integration — already works during development.

2. **Embedded AI** (this spec): An LLM running inside or alongside Nori,
   powering features in Nori's own UI — chat, voice, photo, recipe assist.
   Uses MCP tools to interact with Nori's internals. This is the v2
   integration that makes Nori AI-native for shop floor workers who don't
   use a terminal.

## Where

- Backend: AI service layer that communicates with Ollama
- Frontend: AI prompt UI elements embedded in the execution and recipe views
- Infrastructure: Ollama container in Docker Compose (optional)

## Why

The AI features serve one core purpose: **reduce the friction of documentation
and analysis**. The reason SOPs don't get written, bottlenecks don't get
identified, and processes don't improve is because the human effort required
is too high relative to the immediate payoff.

AI inverts this: documentation happens as a side effect of working. Analysis
happens automatically. The operator's job is to build furniture — Nori's AI
handles the knowledge capture.

Key constraint: **Nori never owns AI costs for other shops.** The default
is local Ollama (zero external cost, full privacy). Shops that want cloud
LLMs bring their own API key (BYOK). This matters for:
- Shop floor privacy (photos of proprietary designs)
- Offline operation (spotty shop wifi — local Ollama still works)
- Cost (no per-token billing from Nori)
- Self-hosting promise (the whole point of Nori)
- Managed hosting viability (subscription covers infra, not AI tokens)

## How

### LLM Provider Configuration (BYOK)

Nori supports multiple LLM backends. Shops choose their own cost/quality
tradeoff:

**Local (Ollama) — Default, recommended for self-hosted:**
- Ollama runs as an optional Docker Compose service
- Models run on the shop's hardware (needs ~8GB RAM for 7B models)
- Zero external API cost, full privacy, works offline
- Best for: shops with a homelab or decent workstation

**Cloud (Bring Your Own Key) — For shops without GPU hardware:**
- Shop provides their own API key for OpenAI, Anthropic, or other providers
- Nori proxies requests to the cloud endpoint
- Higher quality models available (GPT-4, Claude, etc.)
- Best for: cloud-hosted Nori, shops without local GPU

**Disabled — Everything works without AI:**
- All AI features degrade gracefully (see Graceful Degradation below)
- No LLM container needed, no API keys needed
- Best for: shops that just want the task/recipe workflow

Nori communicates with the LLM via a provider-agnostic interface. The
embedded AI features use MCP tools (see mcp-server.md) to query and act
on Nori data, then pass context + tool results to the LLM for generation.

```
User (chat/voice/photo) → Nori AI Service → LLM Provider (Ollama/cloud)
                              ↕                    ↕
                         MCP Tools            Model response
                              ↕
                        Nori Service Layer
```

Model selection is configurable per feature (different models for different
tasks):

Recommended models for Ollama (subject to change as ecosystem evolves):
- **Text generation/summarization**: Llama 3 8B or Mistral 7B (fast, good
  quality for structured tasks)
- **Vision**: LLaVA or Llama 3 Vision (for photo understanding)
- **Transcription**: Whisper (for voice-to-text, if run through Ollama or
  a separate service)

### Feature: First-Time Capture Assistant

When an operator is running a job in first-time capture mode (see
task-execution.md), the AI helps structure the captured data:

1. **Between steps**: "You just completed a 25-minute step. Based on the
   photo you took, it looks like you were fitting mortise and tenon joints.
   Want me to title this step 'Cut and fit mortise-and-tenon joints'?"
2. **After the job**: Takes all captured notes, photos, and timing data and
   generates a structured recipe draft (TOML) with titled steps, instructions,
   and estimated times.
3. **Review prompt**: Presents the draft for human review before saving as a
   RecipeVersion.

### Feature: Recipe Refinement Suggestions

After a job completes (not first-time — normal execution against an existing
recipe):

1. Compare actual task times to recipe estimates across recent executions.
2. Review deviation notes from this and recent executions.
3. Generate suggestions:
   - "Task 4 consistently takes 2x the estimated time. Consider splitting
     it into two tasks or updating the estimate."
   - "3 out of 5 recent operators deviated on Task 7. The instruction may
     need updating."
   - "Task 3 and Task 4 are always completed together in under 5 minutes.
     Consider merging them."

### Feature: Voice-to-Text

For hands-dirty situations where typing is impractical:

1. Operator taps "Voice Note" on the execution UI.
2. Browser records audio (using Web Speech API or MediaRecorder).
3. Audio is sent to Ollama/Whisper for transcription.
4. Transcribed text is attached as a deviation note or recipe annotation.

Fallback: Use the browser's built-in Web Speech API for on-device
transcription (works offline, lower quality but zero latency).

### Feature: Photo Understanding

When an operator attaches a photo to a task:

1. The photo is sent to the vision model.
2. The model generates a description: "Close-up of a mortise-and-tenon joint
   being dry-fitted. The tenon appears slightly loose."
3. This description is suggested as a caption or added to the task or recipe
   instructions.

This is especially useful for first-time capture: instead of the operator
stopping to type what they're looking at, they just snap a photo and the AI
describes it.

### Feature: Bottleneck Summary (Plain Language)

The analytics engine (see bottleneck-analytics.md) produces raw metrics.
The AI layer translates these into actionable summaries:

- "Your shop's biggest constraint this month has been the Joinery station.
  Jobs are waiting an average of 2 days before you can get to them. The
  rest of your stations have plenty of capacity. Consider whether any
  joinery operations could be done at the Assembly station instead, or
  whether it's worth investing in a dedicated mortiser to speed up the
  most time-consuming step."

### Feature: Auto-Tagging

When jobs or tasks are created, the AI suggests tags:
- Job description mentions "sand and spray" → suggest tags: `finish`, `spray`
- Deviation note mentions "glue" → suggest tag: `adhesive`, `assembly`
- Replenishment task for lumber → auto-tag: `restock`, `lumber`

### Graceful Degradation

Every AI feature must work without AI:
- First-time capture: Tasks are captured with manual titles, no auto-structuring
- Recipe refinement: Raw stats shown instead of narrative suggestions
- Voice-to-text: Falls back to typed notes
- Photo understanding: Photos attached without auto-description
- Bottleneck summary: Dashboard shows numbers without narrative
- Auto-tagging: Manual tagging only

The system should never block on AI. All AI calls are async with timeouts.

### Configuration

```yaml
ai:
  enabled: true
  provider: ollama              # ollama | openai | anthropic | disabled
  ollama:
    url: http://ollama:11434
    models:
      text: llama3:8b
      vision: llava:13b
      transcription: whisper
  openai:                       # only used when provider: openai
    api_key: ${NORI_OPENAI_KEY} # env var reference, never stored in plain text
    models:
      text: gpt-4o-mini
      vision: gpt-4o
  anthropic:                    # only used when provider: anthropic
    api_key: ${NORI_ANTHROPIC_KEY}
    models:
      text: claude-sonnet-4-20250514
      vision: claude-sonnet-4-20250514
  features:
    first_time_capture: true
    recipe_refinement: true
    voice_to_text: true
    photo_understanding: true
    bottleneck_summary: true
    auto_tagging: true
```

Environment variables for sensitive values:
- `NORI_AI_PROVIDER` — override provider
- `NORI_OPENAI_KEY` — OpenAI API key
- `NORI_ANTHROPIC_KEY` — Anthropic API key
- `NORI_OLLAMA_URL` — Ollama endpoint

### API Surface

AI features are not directly exposed as API endpoints. They're triggered
internally by other features:
- SOP execution completion → triggers refinement analysis
- Photo upload → triggers vision model
- Analytics query → triggers summary generation

One exception:
```
POST   /api/ai/transcribe                          — Voice-to-text endpoint
POST   /api/ai/describe-image                      — Photo description endpoint
```

## Open Questions

- What's the minimum hardware to run Ollama with a 7B model? (Roughly 8GB
  RAM. Should document requirements clearly.)
- Should AI-generated content be clearly labeled? ("AI suggested: ...")
  Yes — operators should always know what's human vs. machine.
- How do we handle model updates? Auto-pull latest via Ollama, or pin to
  specific versions for reproducibility?
- Should there be a "train on our data" feature where the shop's own SOP
  data fine-tunes the model? (Not for v1, but architecturally interesting.)
- Is Whisper the right transcription path, or should we lean on browser
  APIs exclusively? (Browser API is simpler; Whisper is more accurate.)
- For the managed/hosted offering, should there be a "Nori AI" tier that
  bundles an Ollama instance in the subscription? Or always BYOK?
- Should the BYOK configuration be per-feature? (e.g., use local Ollama for
  auto-tagging but cloud GPT-4 for recipe drafting.) This would let shops
  optimize cost vs. quality per feature.
- How do we handle provider-specific prompt formats? Abstract behind a
  common interface, or use provider SDKs directly?
