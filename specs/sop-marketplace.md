# SOP Marketplace

## Who

- **Shop owners**: Share SOPs they've created, discover SOPs from others.
- **The community**: Small woodworking/manufacturing shops that benefit from
  shared knowledge.
- **Nori project maintainers**: Curate and moderate public SOPs.

## What

A public repository of SOPs that Nori users can browse, fork into their own
Space, and contribute back to. Think of it like GitHub for manufacturing
procedures — you don't need to write an SOP for changing a Dewalt miter saw
blade from scratch because someone already has.

## Where

- Backend: Marketplace API (could be a central hosted service or federated)
- Frontend: Marketplace browse/search page within Nori
- Data model: PublicSOP entities, fork tracking

## Why

From the original requirements:

> "When creating SOPs I also want there to be a 'marketplace' of SOPs.
> If I make an SOP for tuning my jointer, I'd like to make this public so
> other companies can fork my SOP. There's really no need to reinvent the
> wheel. Everyone changes the saw blade on a Dewalt miter saw the same way."

This is the community play for the open source project. Individual SOPs
(specific products) are proprietary, but process SOPs (tool maintenance,
common techniques, safety procedures) are universal. Sharing them:
- Saves every shop from reinventing the wheel
- Creates a network effect that makes Nori more valuable
- Builds community around the project
- Helps new shops get started with battle-tested procedures

## How

### What Gets Shared

**Good candidates for the marketplace:**
- Tool maintenance (jointer tuning, table saw alignment, blade changes)
- Common techniques (mortise and tenon, dovetails, edge banding)
- Safety procedures (dust collection setup, finish ventilation)
- Shop operations (3S procedures, opening/closing checklists)
- Finishing processes (spray schedules, oil application, curing)

**Not shared:**
- Product-specific SOPs (your proprietary dining table design)
- Customer-specific processes
- Anything with proprietary jigs or trade secrets

### Publishing to the Marketplace

1. Owner selects an SOP and clicks "Publish to Marketplace"
2. They choose which version to publish (must be a published version)
3. They add marketplace metadata:
   - Category (tool maintenance, technique, safety, operations, finishing)
   - Tags (jointer, dewalt, hand-tools, spray, etc.)
   - Description (why this SOP is useful)
   - Difficulty level (beginner, intermediate, advanced)
   - Equipment required (tools/machines needed)
4. Personal/proprietary information is stripped or flagged for review
5. SOP is submitted for moderation (or auto-published, depending on
   trust model)

### Forking

When a shop finds a useful marketplace SOP:
1. Click "Fork to My Space"
2. A copy of the SOP is created in their Space as a new SOPTemplate
3. The fork tracks its origin (marketplace SOP ID + version)
4. The shop can modify their fork freely
5. Optionally, they can "sync" with upstream updates (pull new changes
   from the original)

### Discovery

- **Browse by category**: Tool Maintenance, Techniques, Safety, etc.
- **Search**: Full-text search across SOP names, descriptions, step titles
- **Filter**: By equipment required, difficulty, popularity
- **Sort**: Most forked, most recently updated, highest rated
- **Featured**: Curated collection of high-quality SOPs

### Contribution Back

If a shop improves a forked SOP:
1. They can submit their changes as a "suggestion" back to the original
2. The original author reviews and can accept/reject (like a pull request)
3. This creates a virtuous cycle of community improvement

### Trust and Moderation

Options for v1 (decide based on community size):
- **Curated**: Maintainers review and approve all submissions
- **Open with flagging**: Anyone can publish, community can flag issues
- **Reputation-based**: Trusted contributors get auto-publish, new users
  go through review

### Architecture Considerations

**Centralized vs. Federated:**
- **Centralized** (simpler): A hosted marketplace service that all Nori
  instances connect to. Easier to moderate, search, and discover.
- **Federated** (more aligned with self-hosting ethos): Each Nori instance
  can expose its public SOPs, and instances discover each other. Much harder
  to build but more resilient.

For v1: centralized is pragmatic. A simple API that Nori instances call to
browse and fork. Can be a free hosted service run by the Nori project.

### Data Model

```
MarketplaceSOP
  - ID: uuid
  - AuthorID: uuid (Nori user who published)
  - AuthorShopName: string (optional attribution)
  - Name: string
  - Description: text
  - Category: enum
  - Tags: string[]
  - DifficultyLevel: enum (beginner, intermediate, advanced)
  - EquipmentRequired: string[]
  - SOPData: json (serialized SOP version with steps, media references)
  - Version: int (marketplace version, independent of source SOP versioning)
  - ForkCount: int (popularity metric)
  - Rating: decimal (community rating)
  - Status: enum (submitted, approved, published, rejected, archived)
  - CreatedAt, UpdatedAt: timestamp

MarketplaceFork
  - ID: uuid
  - MarketplaceSOPID: uuid
  - SpaceID: uuid (the shop that forked it)
  - SOPTemplateID: int (the local SOP created from the fork)
  - ForkedVersion: int (which marketplace version was forked)
  - CreatedAt: timestamp
```

### API Surface

```
GET    /api/marketplace/sops                       — Browse/search marketplace
GET    /api/marketplace/sops/:id                    — Get marketplace SOP detail
POST   /api/marketplace/sops                       — Publish an SOP to marketplace
POST   /api/marketplace/sops/:id/fork              — Fork to my space
POST   /api/marketplace/sops/:id/suggest            — Submit improvement suggestion
GET    /api/marketplace/categories                  — List categories
```

## Open Questions

- Should the marketplace be a separate hosted service, or built into every
  Nori instance with peer-to-peer discovery? (Centralized for v1.)
- How should media (photos, videos) be handled in marketplace SOPs? Hosted
  centrally? Or references that may break? (Probably need central hosting
  for reliability.)
- What's the intellectual property model? Should there be a license on
  marketplace SOPs? (Creative Commons seems natural.)
- Should shops be able to sell premium SOPs on the marketplace? (Not for v1,
  but could be a sustainability model for the project.)
- How do we prevent low-quality SOPs from flooding the marketplace? Curation
  is expensive at scale.
