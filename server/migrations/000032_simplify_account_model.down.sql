-- --------------------------------------------------------------------
-- Restore Account billing/contact fields
-- --------------------------------------------------------------------

-- Restore sync flag column
ALTER TABLE accounts 
ADD COLUMN sync_contact_and_billing_info BOOLEAN NOT NULL DEFAULT false;

-- Restore contact-related columns
ALTER TABLE accounts 
ADD COLUMN contact_first_name TEXT,
ADD COLUMN contact_last_name TEXT,
ADD COLUMN contact_phone_number TEXT,
ADD COLUMN contact_phone_ext TEXT,
ADD COLUMN contact_email TEXT,
ADD COLUMN contact_address TEXT,
ADD COLUMN contact_city TEXT,
ADD COLUMN contact_state TEXT,
ADD COLUMN contact_zip TEXT;

-- Restore billing-related columns
ALTER TABLE accounts 
ADD COLUMN billing_first_name TEXT,
ADD COLUMN billing_last_name TEXT,
ADD COLUMN billing_phone_number TEXT,
ADD COLUMN billing_phone_ext TEXT,
ADD COLUMN billing_email TEXT,
ADD COLUMN billing_address TEXT,
ADD COLUMN billing_city TEXT,
ADD COLUMN billing_state TEXT,
ADD COLUMN billing_zip TEXT;
