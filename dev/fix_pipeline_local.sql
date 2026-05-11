-- Script ini dijalankan manual via pgAdmin atau psql untuk fix pipeline yang salah config
-- Tujuan: pastikan pipeline forwarder mengarah ke broker LOKAL (emqx dalam docker network)
-- sehingga iot-suhu-be yang subscribe di local broker bisa menerima data dari forwarder

-- Cek pipeline yang ada
SELECT pipeline_id, name, source_topic, broker_url, dest_topic, interval_minutes, is_active
FROM pipelines
ORDER BY pipeline_id;

-- Jika ada pipeline yang broker_url-nya cloud (misal emqx.miegacoan.id), update ke local:
-- UPDATE pipelines 
-- SET broker_url = 'tcp://emqx:1883',   -- service name di docker-compose / internal network
--     interval_minutes = 1,              -- 1 menit untuk dev/simulation (bukan 8 menit)
--     is_active = TRUE
-- WHERE pipeline_id = <ID_PIPELINE_CLOUD>;

-- Atau insert pipeline lokal baru jika belum ada:
INSERT INTO pipelines (name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active)
VALUES (
    'Local-to-Local Forward (DEV)',
    'sensor/data/ingest',
    'tcp://emqx:1883',          -- broker lokal dalam docker network
    'sensor/data/forwarded',
    'apps',                      -- sesuaikan dengan MQTT_USERNAME di .env servfor
    'apps',                      -- sesuaikan dengan MQTT_PASSWORD di .env servfor
    1,                           -- 1 menit untuk dev/testing (bukan 8 menit)
    TRUE
)
ON CONFLICT DO NOTHING;

-- Pastikan tidak ada pipeline cloud aktif yang mengganggu:
-- UPDATE pipelines SET is_active = FALSE WHERE broker_url LIKE '%miegacoan%' OR broker_url LIKE '%emqx.%';
