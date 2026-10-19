-- +goose Up
-- Lab/demo signatures so category rules expand to matchable process names.
-- Production catalogs should be managed via a future Categories admin UI.

INSERT INTO category_signatures (category_code, rule_type, operator, value)
SELECT v.category_code, v.rule_type, v.operator, v.value
FROM (
    VALUES
        ('messaging', 'process_name', 'equals', 'whatsapp.exe'),
        ('messaging', 'process_name', 'equals', 'Telegram.exe'),
        ('messaging', 'process_name', 'equals', 'Discord.exe'),
        ('messaging', 'process_name', 'equals', 'Slack.exe'),
        ('p2p', 'process_name', 'equals', 'utorrent.exe'),
        ('p2p', 'process_name', 'equals', 'qbittorrent.exe'),
        ('p2p', 'process_name', 'equals', 'transmission-gtk'),
        ('game', 'process_name', 'equals', 'steam.exe'),
        ('game', 'process_name', 'equals', 'EpicGamesLauncher.exe'),
        ('remote_desktop', 'process_name', 'equals', 'TeamViewer.exe'),
        ('remote_desktop', 'process_name', 'equals', 'AnyDesk.exe'),
        ('remote_desktop', 'process_name', 'equals', 'mstsc.exe')
) AS v(category_code, rule_type, operator, value)
WHERE NOT EXISTS (
    SELECT 1
    FROM category_signatures cs
    WHERE cs.category_code = v.category_code
      AND cs.rule_type = v.rule_type
      AND cs.operator = v.operator
      AND cs.value = v.value
);

-- +goose Down
DELETE FROM category_signatures
WHERE (category_code, rule_type, operator, value) IN (
    ('messaging', 'process_name', 'equals', 'whatsapp.exe'),
    ('messaging', 'process_name', 'equals', 'Telegram.exe'),
    ('messaging', 'process_name', 'equals', 'Discord.exe'),
    ('messaging', 'process_name', 'equals', 'Slack.exe'),
    ('p2p', 'process_name', 'equals', 'utorrent.exe'),
    ('p2p', 'process_name', 'equals', 'qbittorrent.exe'),
    ('p2p', 'process_name', 'equals', 'transmission-gtk'),
    ('game', 'process_name', 'equals', 'steam.exe'),
    ('game', 'process_name', 'equals', 'EpicGamesLauncher.exe'),
    ('remote_desktop', 'process_name', 'equals', 'TeamViewer.exe'),
    ('remote_desktop', 'process_name', 'equals', 'AnyDesk.exe'),
    ('remote_desktop', 'process_name', 'equals', 'mstsc.exe')
);
