-- --------------------------------------------------------------------
-- Simplify Account model: drop billing/contact fields for v1
-- --------------------------------------------------------------------

-- Drop all contact-related columns
ALTER TABLE accounts 
DROP COLUMN IF EXISTS contact_first_name,
DROP COLUMN IF EXISTS contact_last_name,
DROP COLUMN IF EXISTS contact_phone_number,
DROP COLUMN IF EXISTS contact_phone_ext,
DROP COLUMN IF EXISTS contact_email,
DROP COLUMN IF EXISTS contact_address,
DROP COLUMN IF EXISTS contact_city,
DROP COLUMN IF EXISTS contact_state,
DROP COLUMN IF EXISTS contact_zip;

-- Drop all billing-related columns
ALTER TABLE accounts 
DROP COLUMN IF EXISTS billing_first_name,
DROP COLUMN IF EXISTS billing_last_name,
DROP COLUMN IF EXISTS billing_phone_number,
DROP COLUMN IF EXISTS billing_phone_ext,
DROP COLUMN IF EXISTS billing_email,
DROP COLUMN IF EXISTS billing_address,
DROP COLUMN IF EXISTS billing_city,
DROP COLUMN IF EXISTS billing_state,
DROP COLUMN IF EXISTS billing_zip;

-- Drop the sync flag column
ALTER TABLE accounts 
DROP COLUMN IF EXISTS sync_contact_and_billing_info;
