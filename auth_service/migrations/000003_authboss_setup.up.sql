-- Drop unused Better Auth tables
DROP TABLE IF EXISTS account;
DROP TABLE IF EXISTS session;
DROP TABLE IF EXISTS verification;

-- Add Authboss required columns to users
ALTER TABLE users
ADD COLUMN recover_token VARCHAR(255),
ADD COLUMN recover_token_expiry TIMESTAMP WITH TIME ZONE,
ADD COLUMN confirm_token VARCHAR(255);
