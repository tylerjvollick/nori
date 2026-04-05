# Orders

## Who

- **Shop owners**: Create quotes, confirm orders, track delivery dates.
- **Floor managers**: See incoming work, plan capacity.
- **Operators**: See which customer and order a job belongs to.

## What

Customer and order management — the entry point for work into the shop. Orders
represent customer commitments with due dates. Each order contains line items
that describe what's being built. Confirming an order **pours recipes** to
create Jobs that flow through the shop (see job-flow.md, recipes.md).

## Where

- Backend: Customer, Order, and OrderLineItem models, order API endpoints
- Frontend: Order list, order detail page, quote-to-order flow
- Data model: see data-model.md

## Why

Currently, there's no connection between "a customer wants a table" and "the
shop is building a table." Sales lives in email/spreadsheets, and project
management is separate. Nori closes this gap with a single pipeline from
quote to delivery.

This also enables a critical metric: **quoted lead time vs. actual lead time**.
If you consistently quote 6 weeks but deliver in 8, Nori surfaces the gap.

## How

### Order Lifecycle

```
[Quoted] → [Confirmed] → [In Progress] → [Completed] → [Delivered]
                                                          ↑
                                                    [Cancelled]
```

- **Quoted**: A proposal sent to the customer. No jobs created yet.
- **Confirmed**: Customer accepted. Recipes are poured → Jobs created.
- **In Progress**: At least one job has active tasks.
- **Completed**: All jobs done. Ready for delivery/pickup.
- **Delivered**: Customer has received the product. Order closed.
- **Cancelled**: Order cancelled at any stage.

### Order → Job Creation (Pouring)

When an order is confirmed:

1. For each line item with a linked Recipe, Nori **pours the recipe** to
   create a Job (root Task) with its full task subgraph.
2. Recipe variables are populated from the line item (wood species, dimensions,
   finish type, etc.).
3. If quantity is 1: one Job.
4. If quantity > 1: the user chooses:
   - **Per-unit**: N separate Jobs (one per item). Better for TOC flow
     visibility. The recipe's `batch_size` variable is set to 1.
   - **Batch**: One Job with a loop variable `batch_size=N`. Tasks repeat
     per unit within a single job.
5. Each Job references the Customer and has the order's due date.
6. Jobs enter the ready-work system. Tasks at the first station appear in
   the ready queue.

### Customer Model

Simple for v1. Not a full CRM.

```
Customer
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - Name: string (required)
  - Email: string (nullable)
  - Phone: string (nullable)
  - Address: text (nullable)
  - Notes: text (nullable)
  - CreatedAt, UpdatedAt: timestamp
```

### Order Model

```
Order
  - ID: uuid
  - SpaceID: uuid (FK → Space)
  - CustomerID: uuid (FK → Customer)
  - OrderNumber: string (auto-generated, e.g., "ORD-2026-042")
  - Status: enum (quoted, confirmed, in_progress, completed, delivered, cancelled)
  - DueDate: timestamp (nullable — when customer expects delivery)
  - Notes: text (nullable — special instructions)
  - QuotedAt: timestamp (nullable)
  - ConfirmedAt: timestamp (nullable)
  - CompletedAt: timestamp (nullable)
  - DeliveredAt: timestamp (nullable)
  - CreatedByID: uuid (FK → User)
  - CreatedAt, UpdatedAt: timestamp
```

### OrderLineItem Model

```
OrderLineItem
  - ID: uuid
  - OrderID: uuid (FK → Order)
  - RecipeID: uuid (FK → Recipe, nullable — what product type)
  - Description: string ("72x36 Walnut Dining Table with breadboard ends")
  - Quantity: int
  - UnitPrice: decimal (nullable — for quote generation)
  - Notes: text (nullable — customization details)
  - Vars: jsonb (nullable — recipe variable overrides: {"wood_species": "Walnut"})
  - DisplayOrder: int
  - CreatedAt, UpdatedAt: timestamp
```

The `Vars` field maps to recipe variables. When the order is confirmed and
recipes are poured, these values are passed to the formula engine.

### Lead Time Tracking

Two key metrics computed from order + job data:
- **Quoted lead time**: DueDate - ConfirmedAt
- **Actual lead time**: CompletedAt - ConfirmedAt
- **Delta**: Actual - Quoted (positive = late, negative = early)

Over time, this surfaces systemic quoting problems. If dining tables are
always 2 weeks late but cutting boards are on time, the constraint is visible.

### Order UI

**Order List**:
- Sortable by status, due date, customer
- Color-coded status badges
- Overdue orders highlighted in red
- Click through to detail

**Order Detail**:
- Customer info
- Line items with linked recipes
- Jobs created from each line item with current progress
- Timeline: key dates (quoted, confirmed, started, completed, delivered)
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
POST   /api/orders/:id/confirm                     — Confirm → pour recipes → create jobs
POST   /api/orders/:id/complete                    — Mark completed
POST   /api/orders/:id/deliver                     — Mark delivered

POST   /api/orders/:id/line-items                  — Add line item
PUT    /api/order-line-items/:id                   — Update line item
DELETE /api/order-line-items/:id                   — Remove line item
```

## Open Questions

- Should Nori support generating PDF quotes? Nice to have but scope creep
  for v1. Most small shops use separate invoicing software.

- How should repeat orders work? A "reorder" button that clones line items
  from a previous order, pours fresh jobs? Probably yes — common use case.

- Should pricing/invoicing be in scope? Leaning toward basic pricing on line
  items for quotes, but no invoicing or payment processing. Leave that to
  QuickBooks/Wave.

- Should Order be a first-class entity or just a Task with type `order`?
  Keeping it separate for now because orders have lifecycle states and line
  items that don't map cleanly to the Task model. But this could be
  reconsidered.
