DROP INDEX IF EXISTS idx_messages_conversation;
DROP INDEX IF EXISTS idx_follow_up_reminders_workspace;
DROP INDEX IF EXISTS idx_internal_notes_workspace;
DROP INDEX IF EXISTS idx_message_templates_workspace;
DROP INDEX IF EXISTS idx_conversations_workspace;
DROP INDEX IF EXISTS idx_channels_workspace;
DROP INDEX IF EXISTS idx_leads_workspace;

DROP TABLE IF EXISTS follow_up_reminders;
DROP TABLE IF EXISTS internal_notes;
DROP TABLE IF EXISTS message_templates;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS leads;

DROP EXTENSION IF EXISTS "uuid-ossp";
