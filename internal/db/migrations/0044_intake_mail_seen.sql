-- Every mail the intake has taken from a mailbox, by Message-ID. A message is
-- only flagged read on the server after this row exists, and this row is
-- checked before anything is created, so a crash between the two can at most
-- fetch a mail twice — never file it twice.
CREATE TABLE intake_mail_seen (
    org_key TEXT NOT NULL,
    message_id TEXT NOT NULL,
    intake_id TEXT NOT NULL DEFAULT '',
    seen_at TEXT NOT NULL,
    PRIMARY KEY (org_key, message_id)
);
