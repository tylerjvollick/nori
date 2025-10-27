-- Create accounts table
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NULL,
    sync_contact_and_billing_info BOOLEAN NOT NULL,
    
    contact_first_name TEXT NULL,
    contact_last_name TEXT NULL,
    contact_phone_number TEXT NULL,
    contact_phone_ext TEXT NULL,
    contact_email TEXT NULL,
    contact_address TEXT NULL,
    contact_city TEXT NULL,
    contact_state TEXT NULL,
    contact_zip TEXT NULL,

    billing_first_name TEXT NULL,
    billing_last_name TEXT NULL,
    billing_phone_number TEXT NULL,
    billing_phone_ext TEXT NULL,
    billing_email TEXT NULL,
    billing_address TEXT NULL,
    billing_city TEXT NULL,
    billing_state TEXT NULL,
    billing_zip TEXT NULL,

    plan TEXT DEFAULT 'trial',

    created_by_user_id UUID NOT NULL,
    CONSTRAINT fk_accounts_created_by_user
        FOREIGN KEY(created_by_user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

