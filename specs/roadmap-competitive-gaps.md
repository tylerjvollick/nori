# Roadmap: Competitive Gap Features

This document captures features identified through a cross-analysis of Nori
against Odoo (~46 modules) and ERPNext (~13 modules). These are capabilities
those ERPs offer that Nori currently lacks, evaluated through the lens of
**what actually matters for a 1-5 person custom manufacturing shop**.

Features are organized into tiers. Tier 1 features are natural extensions of
Nori's existing design. Tier 2 features are worth considering as the product
matures. Tier 3 features belong to the ERPs — Nori should integrate rather
than reimplement.

This is a living document. As the business grows and real needs become clear,
features can be promoted between tiers or dropped entirely.

---

## Tier 1 — Natural Extensions

High value for Nori's target user. These fill real gaps in the daily workflow
and build on features Nori already has or plans to have.

### Simple Invoicing

**What:** Generate invoices from completed orders. Track paid/unpaid status.
Export to PDF. Not full accounting — just closing the order lifecycle.

**Why it matters:** Every shop needs to invoice customers. Today they
context-switch to QuickBooks, Wave, or a spreadsheet. If Nori tracks the order
from intake to completion but can't produce an invoice, the workflow is always
split. This is the single highest-value gap — it keeps users inside Nori for
the full order lifecycle.

**Relates to:** Orders spec (order completion triggers invoice generation),
Materials & BOM (material costs feed line items), Time Tracking (labor costs
from logged hours).

**Build or integrate:** Build a minimal native version (PDF generation, payment
status tracking). Provide integration hooks for QuickBooks/Xero for shops that
want proper accounting.

**Complexity:** Medium

**ERP equivalent:** Odoo Invoicing/Accounting, ERPNext Accounting (Sales Invoice)

---

### Simple Purchasing / Reorder

**What:** Generate purchase orders from material reorder signals. Even just a
PDF to email to a supplier. Track PO status (sent, received, partial).

**Why it matters:** Nori already plans pull-signal reorder (material drops below
threshold -> replenishment job created on the flow board). Extending this to
produce an actual purchase order closes the procurement loop. Without it, the
operator sees "reorder walnut" on the board but still has to manually email
the lumber yard.

**Relates to:** Materials & BOM spec (reorder thresholds, pull signals),
Job Flow (replenishment jobs).

**Build or integrate:** Build minimal native version (PO generation, supplier
contact list, receive-against-PO stock adjustment). Keep it simple — no
multi-level approvals, no supplier scorecards.

**Complexity:** Medium

**ERP equivalent:** Odoo Purchase, ERPNext Procurement

---

### Equipment Maintenance

**What:** Preventive and reactive maintenance scheduling per station/tool.
Track maintenance history, schedule recurring tasks, log breakdowns.

**Why it matters:** Small shops depend on their tools. A CNC router or
table saw going down halts production. Nori already models stations as
first-class entities — maintenance is a natural extension. It ties directly
into Theory of Constraints: downtime on the bottleneck station is the most
expensive downtime in the shop.

**Relates to:** Stations spec (each station/tool gets a maintenance schedule),
Job Flow (maintenance jobs appear on the flow board alongside production jobs),
Bottleneck Analytics (maintenance downtime factors into constraint scoring).

**Build or integrate:** Build native. This is inherently shop-floor-specific
and doesn't exist well in external tools.

**Complexity:** Low-Medium

**ERP equivalent:** Odoo Maintenance, ERPNext Assets (maintenance tracking)

---

### Knowledge Base

**What:** General shop-level documentation that isn't tied to a specific SOP.
Safety procedures, supplier contacts, finishing recipes, tool setup notes,
shop policies, material handling guides.

**Why it matters:** SOPs cover *how to build specific products*. But shops also
need a place for general knowledge: "What's our return policy?", "How do I
adjust the bandsaw fence?", "Which finish do we use on walnut vs. maple?"
Today this lives in someone's head, a binder, or scattered Google Docs.

**Relates to:** SOP Authoring (shared infrastructure — rich text, media
attachments, versioning), SOP Marketplace (knowledge base articles could be
shareable too), AI Features (AI can help draft/organize knowledge base content).

**Build or integrate:** Build native. Reuse the same content infrastructure
as SOPs (rich text + media). Tag-based organization.

**Complexity:** Low

**ERP equivalent:** Odoo Knowledge, ERPNext Knowledge Base (in Support module)

---

## Tier 2 — Worth Considering for v2+

Medium value. These would differentiate Nori further but aren't essential for
the core manufacturing workflow. Evaluate as the product matures and real user
needs become clearer.

### Quoting / Estimating from Historical Data

**What:** Generate customer quotes by pulling from historical SOP time data
and BOM material costs. "Last time you built this style of table it took 14
hours and $180 in materials — suggested quote: $X at your target margin."

**Why it matters:** Custom furniture shops spend significant time quoting, often
from gut feel. They frequently underquote and lose money. Nori, with its
automatic time capture and BOM tracking, is *uniquely positioned* to solve this.
Neither Odoo nor ERPNext can do data-driven quoting for custom/one-off work
because they don't capture execution time at the SOP step level.

**Relates to:** Time Tracking (historical step times), Materials & BOM
(material costs), Orders (quote-to-order conversion), Bottleneck Analytics
(capacity-aware delivery date estimation).

**Build or integrate:** Build native. This is Nori's secret weapon — the data
is already being captured through normal SOP execution.

**Complexity:** Medium-High

**ERP equivalent:** Odoo Sales (quotations), ERPNext Sales (quotations). But
neither connects quotes to actual historical production data the way Nori could.

---

### Customer Portal

**What:** A read-only view for customers to check their order status. "Your
dining table is at the Finish station (step 5 of 7). Estimated completion:
next Tuesday."

**Why it matters:** For shops doing $5K-$50K custom pieces, customers want
visibility. Today they call or email the shop, interrupting production. A
simple status page (even just a magic link per order) saves time and builds
trust. It's a professional differentiator for small shops.

**Relates to:** Orders (order status), Job Flow (current station/step),
Time Tracking (estimated completion from historical data), Auth & Tenancy
(customer-scoped read-only access).

**Build or integrate:** Build native. Simple implementation — a public or
magic-link-protected page showing order progress. No customer login system
needed for v1.

**Complexity:** Medium

**ERP equivalent:** Odoo Website/Portal, ERPNext Customer Portal

---

### Quality Inspection Checklists

**What:** Formal QC checkpoints with pass/fail criteria and photo
documentation. Particularly valuable as a pre-ship gate.

**Why it matters:** Nori's SOP deviation capture covers in-process quality, but
a formal QC step (especially before shipping) with structured pass/fail
criteria catches problems before they reach the customer. For high-value custom
work, this protects reputation and reduces costly rework after delivery.

**Relates to:** SOP Execution (QC checkpoints can be special SOP step types),
Stations (QC is already a default station), Job Flow (QC as a gate before the
Done column).

**Build or integrate:** Build native. Extend SOP step types to include
"inspection" steps with pass/fail fields, measurement fields, and required
photo documentation.

**Complexity:** Low-Medium

**ERP equivalent:** Odoo Quality, ERPNext Quality (inspections, non-conformance
reporting)

---

### Simple Reporting / Dashboard Builder

**What:** Let users build custom views beyond bottleneck analytics. Revenue by
month, jobs completed per operator, material cost trends, on-time delivery
rate over time.

**Why it matters:** Bottleneck analytics covers the TOC-specific metrics, but
shop owners also need general business visibility. "How much walnut did we use
this quarter?" "What's our average lead time trending toward?" Today this lives
in spreadsheets.

**Relates to:** Bottleneck Analytics (shares the same underlying data), Time
Tracking (labor data), Materials & BOM (material consumption data), Orders
(revenue and delivery data).

**Build or integrate:** Start with a set of pre-built reports covering the
most common questions. Consider a simple dashboard builder later. For advanced
BI, integrate with external tools (Metabase, Grafana) via the database or API.

**Complexity:** Medium

**ERP equivalent:** Odoo Spreadsheet (BI), ERPNext Report Builder / Query Report

---

### Document Management

**What:** Attach drawings, CAD files, finish samples, customer reference photos,
and design specs to orders, SOPs, and customers. Organized file storage with
tagging.

**Why it matters:** Custom shops constantly reference drawings, customer photos
("I want it to look like this"), and design files. Nori already has media on
SOP steps — extending this to a general file store per ticket/customer
provides a single place for all project-related documents.

**Relates to:** SOP Authoring (already has media infrastructure — TUS chunked
upload, SOPStepMedia), Orders/Tickets (file attachments per ticket), Customers
(reference photos, design files).

**Build or integrate:** Build native. Extend the existing media infrastructure
to support general file attachments on any entity. Tag-based organization.

**Complexity:** Low

**ERP equivalent:** Odoo Documents, ERPNext File Manager

---

## Tier 3 — Leave to the ERPs (Integrate, Don't Build)

These are core ERP modules that are out of scope for Nori. Building them would
dilute focus and compete poorly against mature, full-featured implementations.
Instead, provide clean integration points where it makes sense.

| Module | Why Skip | Integration Opportunity |
|---|---|---|
| **Full Accounting / GL** | Massive scope. QuickBooks, Xero, and Wave exist. Nori's value is on the shop floor, not the balance sheet. | Export invoice data to QuickBooks/Xero. Sync payment status back. |
| **HR / Payroll** | A 1-5 person shop doesn't need an HR module. They use Gusto, ADP, or a payroll service. | Export time tracking data as payroll-ready reports. |
| **CRM / Lead Pipeline** | Custom shops get leads through word of mouth, Instagram, and local reputation. A sales pipeline adds overhead without value at this scale. | Nori's Customer model covers what's needed. No integration required. |
| **eCommerce / Website** | Not relevant for custom/bespoke manufacturing. Shops use Squarespace, Shopify, or Instagram for their web presence. | Potential Shopify webhook: new Shopify order -> create Nori ticket. |
| **Email / SMS / Social Marketing** | Completely out of scope for a manufacturing operations tool. | None needed. |
| **Point of Sale** | Nori's users don't operate retail storefronts. They build custom pieces for specific customers. | None needed. |
| **Recruitment / Appraisals** | Not relevant at 1-5 person scale. | None needed. |
| **Fleet Management** | Not relevant for a workshop. | None needed. |
| **Subscriptions / Rental** | Not the business model for custom manufacturing. | None needed. |
| **Field Service** | Custom furniture is built in the shop, not on-site (usually). | None needed unless install/delivery tracking becomes a need. |

---

## Integration Strategy

Rather than building ERP modules, Nori should provide clean integration points:

1. **REST API** (already planned) — the foundation for all integrations
2. **Webhooks** — notify external systems on key events (order completed,
   invoice generated, stock low)
3. **QuickBooks/Xero sync** — push invoices, pull payment status
4. **Shopify/WooCommerce webhook** — inbound orders create Nori tickets
5. **CSV/PDF export** — for systems that don't have API integration (payroll
   services, tax prep, etc.)
6. **MCP server** (already planned) — enables LLM-mediated integration
   between Nori and any other tool the user exposes via MCP

The principle: Nori owns the shop floor. Other tools own their domains. Clean
boundaries with well-defined data exchange.

---

## How to Use This Document

This is a roadmap, not a commitment. As Nori grows and real user needs emerge:

1. **Promote features** from Tier 2 to Tier 1 when users ask for them
2. **Drop features** that turn out to be unnecessary
3. **Add new features** discovered through real usage
4. **Create full spec files** (`specs/{name}.md`) when a feature is ready
   to be designed and built — don't implement directly from this document

When a feature from this roadmap gets promoted to a full spec, add it to the
main spec table in `specs/readme.md` and link back to this document for
context on why it was added.
