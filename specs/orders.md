# Orders

## Who

- **Shop owners**: Create quotes, confirm orders, track delivery dates.
- **Floor managers**: See incoming work, plan capacity.
- **Operators**: See which customer and order a job belongs to.

## What

Customer and order management — the entry point for work into the shop. Orders
represent customer commitments with due dates. Each order contains line items
that describe what's being built. Line items spawn Jobs that flow through the
shop (see job-flow.md).

## Where

- Backend: Customer and Order models, order API endpoints
- Frontend: Order list, order detail page, quote-to-order flow
- Data model: see data-model.md

## Why

Currently, there's no connection between "a customer wants a table" and "the
shop is building a table." Sales lives in email/spreadsheets, and project
management lives in Jira. Nori closes this gap with a single pipeline from
quote to delivery.

This also enables a critical metric: **quoted lead time vs. actual lead time**.
If you consistently quote 6 weeks but deliver in 8, you have a systemic
problem that Nori can surface.

## How

### Order Lifecycle

```
[Quoted] → [Confirmed] → [In Progress] → [Completed]
                                              ↓
                                         [Delivered]
```

- **Quoted**: A proposal sent to the customer. Not yet committed. No jobs
  created.
- **Confirmed**: Customer accepted. Jobs are created from line items and
  enter the order queue (first station on the flow board).
- **In Progress**: At least one job from this order has started.
- **Completed**: All jobs finished. Ready for delivery/pickup.
- **Delivered**: Customer has received the product. Order is closed.
- **Cancelled**: Order was cancelled at any stage.

### Order → Job Creation

When an order is confirmed:
1. For each line item, Nori creates Job(s) linked to the line item's
   SOPTemplate.
2. If quantity is 1, one Job is created.
3. If quantity > 1, the user chooses: one Job for the batch, or one Job
   per unit. (Batch is common for identical items like dining chairs;
   per-unit is better for TOC flow visibility.)
4. Each Job snapshots the current published SOPTemplateVersion.
5. Jobs enter the first station's queue (or a special "Order Queue" holding
   area before being released to the floor — the "rope" in drum-buffer-rope).

### Customer Model

Simple for v1. Not a full CRM — just enough to associate orders with people.

```
Customer
  - Name (required)
  - Email, Phone, Address (optional)
  - Notes (free text)
```

Customers are scoped to a Space. A customer can have multiple orders.

### Order Model

```
Order
  - OrderNumber: auto-generated, human-readable (e.g., "ORD-2026-042")
  - Customer: FK
  - Status: enum
  - DueDate: when the customer expects delivery
  - QuotedAt / ConfirmedAt / CompletedAt: lifecycle timestamps
  - Notes: free text for special instructions
```

### OrderLineItem Model

```
OrderLineItem
  - Order: FK
  - SOPTemplate: FK (what product type — e.g., "Walnut Dining Table")
  - Description: human-readable ("72x36 with breadboard ends")
  - Quantity: int
  - UnitPrice: decimal (optional — for quote generation)
  - Notes: customization details
```

### Lead Time Tracking

Two key metrics computed from order data:
- **Quoted lead time**: DueDate - ConfirmedAt
- **Actual lead time**: CompletedAt - ConfirmedAt
- **Delta**: Actual - Quoted (positive = late, negative = early)

Over time, this surfaces systemic quoting problems. If you're always 2 weeks
late on dining tables but on time for cutting boards, that tells you something
about where the constraint is.

### Order List UI

- Sortable by status, due date, customer
- Color-coded status badges
- At-a-glance: overdue orders highlighted in red
- Click through to see all jobs and their progress

### Order Detail UI

- Customer info
- Line items with linked SOPs
- Jobs spawned from each line item with current station/status
- Timeline: key dates (quoted, confirmed, started, completed)
- Actual vs. quoted lead time comparison

### API Surface

```
GET    /api/spaces/:spaceId/customers              — List customers
POST   /api/spaces/:spaceId/customers              — Create customer
PUT    /api/customers/:id                          — Update customer

GET    /api/spaces/:spaceId/orders                 — List orders (filterable)
POST   /api/spaces/:spaceId/orders                 — Create order (quote)
GET    /api/orders/:id                             — Get order with line items + jobs
PUT    /api/orders/:id                             — Update order
POST   /api/orders/:id/confirm                     — Confirm → create jobs
POST   /api/orders/:id/complete                    — Mark completed

POST   /api/orders/:id/line-items                  — Add line item
PUT    /api/order-line-items/:id                   — Update line item
DELETE /api/order-line-items/:id                   — Remove line item
```

## Open Questions

- Should Nori support generating PDF quotes from orders? (Nice to have but
  scope creep for v1.)
- Should there be a "Delivered" status separate from "Completed"? (Completed
  = built, Delivered = in customer's hands. Probably yes — delivery logistics
  is a real concern for furniture.)
- How should repeat orders work? ("Same table as last time.") Should there be
  a "reorder" button that clones line items from a previous order?
- Should pricing/invoicing be in scope at all, or is that better left to
  QuickBooks/Wave/etc.? (Leaning toward basic pricing on line items for quotes,
  but no invoicing or payment processing.)
