-- Drop trigger
DROP TRIGGER IF EXISTS update_webhooks_updated_at ON webhooks;

-- Drop tables
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
