package seed

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)


func SeedData(DB *sql.DB) {

	ckSeedCommands := []string{
		"INSERT INTO ck (ck_id, name) VALUES (1, 'CK 1') ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO ck (ck_id, name) VALUES (2, 'CK 2') ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO ck (ck_id, name) VALUES (3, 'CK 3') ON CONFLICT (name) DO NOTHING;",
	}

	areaSeedCommands := []string{
		"INSERT INTO area (area_id, name, ck_id) VALUES (1, 'Repacking Meat & Pawn', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (2, 'Meat/Pawn Storage', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (3, 'Chili/Mushroom Storage', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (4, 'Frozen Chili & Mushroom', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (5, 'Ambient WH', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (6, 'Packing Storage', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (7, 'Intermediate Room', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (8, 'Corridor Room', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (9, 'Meat Processing', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (10, 'Chilled Room', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (11, 'Metos Room', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (12, 'Wheat Flour Storage', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (13, 'Line Production', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (14, 'IQF', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (15, 'Secondary Packing', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (16, 'Intermediate Packing', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (17, 'Frozen Storage FG', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (18, 'Warehouse Dry FG', 3) ON CONFLICT (name) DO NOTHING;",
		"INSERT INTO area (area_id, name, ck_id) VALUES (19, 'Loading Platform', 3) ON CONFLICT (name) DO NOTHING;",
	}

	doorSeedCommands := []string{
		"INSERT INTO door (door_id, name, area_id) VALUES (11, 'Area In Repacking Meat & Pawn I', 1) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (13, 'Area In Repacking Meat & Pawn II', 1) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (15, 'Area In Repacking Meat & Pawn III', 1) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (17, 'Area In Repacking Meat & Pawn IV', 1) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (21, 'Area Meat/Pawn Storage I', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (23, 'Area Meat/Pawn Storage II', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (25, 'Area Meat/Pawn Storage III', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (22, 'Area Meat/Pawn Storage IV', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (24, 'Area Meat/Pawn Storage V', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (26, 'Area Meat/Pawn Storage VI', 2) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (31, 'Area RM Chilled Room Storage I', 3) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (32, 'Area RM Chilled Room Storage II', 3) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (51, 'Area Ambient WH I', 5) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (52, 'Area Ambient WH II', 5) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (61, 'Area Packing Storage I', 6) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (62, 'Area Packing Storage II', 6) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (101, 'Area Chilled Room IN', 10) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (102, 'Area Chilled Room OUT I', 10) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (104, 'Area Chilled Room OUT II', 10) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (112, 'Area Metos Room OUT I', 11) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (114, 'Area Metos Room OUT II', 11) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (121, 'Area Wheat Flour Storage', 12) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (181, 'Area Ambient WH FG', 18) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (182, 'Area Ambient WH FG', 18) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (171, 'Area Frozen Storage FG IN I', 17) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (173, 'Area Frozen Storage FG IN II', 17) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (172, 'Area Frozen Storage FG OUT I', 17) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (174, 'Area Frozen Storage FG OUT II', 17) ON CONFLICT (door_id) DO NOTHING;",
		"INSERT INTO door (door_id, name, area_id) VALUES (176, 'Area Frozen Storage FG PUT III', 17) ON CONFLICT (door_id) DO NOTHING;",
	}

	adminPasswordHash := "$2a$12$YO.xqo.hJeAB5hki6QGJEuJiSR.1gFFkpOHX7o2wwCXczxQNlu8si"
	userSeedCommands := []string{
		fmt.Sprintf("INSERT INTO users (user_id, username, password_hash) VALUES (1, 'admin', '%s') ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash;", adminPasswordHash),
	}

	// =====================================================================
	// Pipeline seed: pastikan ada pipeline lokal yang forward
	// sensor/data/ingest (CSV) → sensor/data/forwarded (JSON) di broker LOKAL
	//
	// PENTING:
	// - broker_url diambil dari MQTT_BROKER_PUB (.env) = tcp://emqx:1883 di docker
	// - credentials pakai MQTT_USERNAME / MQTT_PASSWORD (bukan _FOR yang untuk cloud)
	//   karena docker-compose EMQX hanya daftarkan user dari MQTT_USERNAME/MQTT_PASSWORD
	// - ON CONFLICT DO UPDATE → pipeline otomatis dikoreksi setiap docker-compose up --build
	//   meski DB sudah punya pipeline lama yang mengarah ke cloud
	// =====================================================================
	localBrokerURL := os.Getenv("MQTT_BROKER_PUB")
	if localBrokerURL == "" {
		localBrokerURL = "tcp://localhost:1883"
	}
	// Gunakan kredensial broker LOKAL (bukan _FOR yang untuk cloud destination)
	localUsername := os.Getenv("MQTT_USERNAME")
	if localUsername == "" {
		localUsername = "apps"
	}
	localPassword := os.Getenv("MQTT_PASSWORD")
	if localPassword == "" {
		localPassword = "apps"
	}

	pipelineSeedCommands := []string{
		// Pipeline 1: Lokal
		fmt.Sprintf(
			`INSERT INTO pipelines (pipeline_id, name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active)
			VALUES (1, 'Local-to-Local Forward', 'sensor/data/ingest', '%s', 'sensor/data/forwarded', '%s', '%s', 1, TRUE)
			ON CONFLICT (pipeline_id) DO UPDATE SET
				broker_url       = EXCLUDED.broker_url,
				username         = EXCLUDED.username,
				password         = EXCLUDED.password,
				interval_minutes = EXCLUDED.interval_minutes,
				is_active        = EXCLUDED.is_active;`,
			localBrokerURL, localUsername, localPassword,
		),
		// Pipeline 2: Cloud (Miegacoan)
		fmt.Sprintf(
			`INSERT INTO pipelines (pipeline_id, name, source_topic, broker_url, dest_topic, username, password, interval_minutes, is_active)
			VALUES (2, 'Cloud Forward (Miegacoan)', 'sensor/data/ingest', 'tcp://emqx.miegacoan.id:1883', 'sensor/data/forwarded', 'saiful', 'saiful123', 8, TRUE)
			ON CONFLICT (pipeline_id) DO NOTHING;`,
		),
	}



	seedCommandList := []struct {
		name     string
		commands []string
	}{
		{"CK", ckSeedCommands},
		{"Area", areaSeedCommands},
		{"Door", doorSeedCommands},
		{"User", userSeedCommands},
		{"Pipeline", pipelineSeedCommands},
	}


	for _, seedGroup := range seedCommandList {

		tx, err := DB.Begin()
		if err != nil {
			log.Printf("Error starting transaction for %s seeding: %v", seedGroup.name, err)
			continue
		}
		for _, command := range seedGroup.commands {
			_, err := tx.Exec(command)
			if err != nil {
				tx.Rollback()
				log.Printf("Error seeding %s data with command '%s': %v. Rolled back transaction.", seedGroup.name, command, err)
				goto nextGroup
			}
		}
		err = tx.Commit()
		if err != nil {
			log.Printf("Error committing transaction for %s seeding: %v", seedGroup.name, err)
		} else {

		}
	nextGroup:
	}
}
