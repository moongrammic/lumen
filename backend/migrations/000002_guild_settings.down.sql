DROP INDEX IF EXISTS idx_messages_channel_id;
DROP INDEX IF EXISTS idx_channels_guild_id;

ALTER TABLE guilds DROP COLUMN IF EXISTS description;
ALTER TABLE guilds DROP COLUMN IF EXISTS icon_url;
