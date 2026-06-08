package core

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func InitMQTT() {
	hst := GetEnv("MQTT_BROKER", "")
	if hst == "" {
		Ftl("MQTT_BROKER environment variable is required")
	}
	usr := GetEnv("MQTT_USER", "apps")
	pwd := GetEnv("MQTT_PASS", "apps")

	opt := mqtt.NewClientOptions()
	opt.AddBroker(hst)
	opt.SetClientID(fmt.Sprintf("web-%d", time.Now().Unix()))
	if usr != "" {
		opt.SetUsername(usr)
	}
	if pwd != "" {
		opt.SetPassword(pwd)
	}
	opt.AutoReconnect = true
	opt.OnConnect = func(cln mqtt.Client) {
		Lg("MQTT connected to %s", hst)
		if tkn := cln.Subscribe("nodes/+/status", 0, mqttHandler); tkn.Wait() && tkn.Error() != nil {
			HndlErr("subscribe status", tkn.Error())
		}
		if tkn := cln.Subscribe("nodes/+/monitor", 0, mqttHandler); tkn.Wait() && tkn.Error() != nil {
			HndlErr("subscribe nodes/+/monitor", tkn.Error())
		}
		if tkn := cln.Subscribe("+/monitor", 0, mqttHandler); tkn.Wait() && tkn.Error() != nil {
			HndlErr("subscribe +/monitor", tkn.Error())
		}
		if tkn := cln.Subscribe("nodes/+/log", 0, mqttHandler); tkn.Wait() && tkn.Error() != nil {
			HndlErr("subscribe nodes/+/log", tkn.Error())
		}
	}
	opt.OnConnectionLost = func(cln mqtt.Client, err error) {
		HndlErr("MQTT lost", err)
	}

	MqttClient = mqtt.NewClient(opt)
	for {
		if tkn := MqttClient.Connect(); tkn.Wait() && tkn.Error() == nil {
			break
		} else {
			Lg("waiting for mqtt broker, retry in 2s...")
			time.Sleep(2 * time.Second)
		}
	}
}

func mqttHandler(cln mqtt.Client, psn mqtt.Message) {
	tpc := psn.Topic()
	prts := strings.Split(tpc, "/")
	if len(prts) < 2 {
		return
	}

	var nodeID, sub string
	if prts[0] == "nodes" && len(prts) >= 3 {
		nodeID = prts[1]
		sub = prts[2]
	} else if len(prts) >= 2 {
		nodeID = prts[0]
		sub = prts[1]
	} else {
		return
	}

	raw := psn.Payload()
	now := time.Now().Format("2006-01-02 15:04:05")

	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	if _, ok := NodeStatus[nodeID]; !ok {
		newMacMatches := MacRegex.FindAllString(nodeID, -1)
		if len(newMacMatches) > 0 {
			newMac := newMacMatches[len(newMacMatches)-1]
			for oldID, oldInfo := range NodeStatus {
				if oldID == nodeID {
					continue
				}
				oldMacMatches := MacRegex.FindAllString(oldID, -1)
				if len(oldMacMatches) > 0 && oldMacMatches[len(oldMacMatches)-1] == newMac {
					Lg("MAC match! Migrating '%s' -> '%s'", oldID, nodeID)
					newNodeInfo := &NodeInfo{
						Ck:           oldInfo.Ck,
						Area:         oldInfo.Area,
						No:           oldInfo.No,
						MinT1:        oldInfo.MinT1,
						MaxT1:        oldInfo.MaxT1,
						MinT2:        oldInfo.MinT2,
						MaxT2:        oldInfo.MaxT2,
						MinT3:        oldInfo.MinT3,
						MaxT3:        oldInfo.MaxT3,
						SHTSuhuMin:   oldInfo.SHTSuhuMin,
						SHTSuhuMax:   oldInfo.SHTSuhuMax,
						SHTHumMin:    oldInfo.SHTHumMin,
						SHTHumMax:    oldInfo.SHTHumMax,
						Interval:     oldInfo.Interval,
						Prefix:       oldInfo.Prefix,
						Model:        oldInfo.Model,
						Version:      oldInfo.Version,
						AppMode:      oldInfo.AppMode,
						Trans:        oldInfo.Trans,
						PassCode:     oldInfo.PassCode,
						IP:           oldInfo.IP,
						RamFreeBytes: oldInfo.RamFreeBytes,
						SD_OK:        oldInfo.SD_OK,
						Logs:         oldInfo.Logs,
						Status:       "online",
						Updated:      now,
					}
					NodeStatus[nodeID] = newNodeInfo
					delete(NodeStatus, oldID)
					break
				}
			}
		}
	}

	if _, ok := NodeStatus[nodeID]; !ok {
		if len(NodeStatus) >= 1000 {
			Lg("[MQTT] Node limit reached, dropping %s", nodeID)
			return
		}
		NodeStatus[nodeID] = &NodeInfo{}
	}
	info := NodeStatus[nodeID]

	switch sub {
	case "status":
		rawStr := string(raw)
		if strings.HasPrefix(rawStr, "state=") || strings.HasPrefix(rawStr, "cmd=") {
			values, err := url.ParseQuery(rawStr)
			if err != nil {
				HndlErr(fmt.Sprintf("[MQTT] Warning: URL decode failed for %s, parsing anyway", nodeID), err)
			}
			boolFields := map[string]bool{"alarm_enabled": true, "sd_ok": true, "relay": true}
			if values != nil {
				m := make(map[string]interface{})
				for k, v := range values {
					if len(v) > 0 {
						if boolFields[k] {
							m[k] = v[0] == "true" || v[0] == "1"
						} else if f, err := strconv.ParseFloat(v[0], 64); err == nil {
							m[k] = f
						} else {
							m[k] = v[0]
						}
					}
				}
				info.FullConfig = m
				b, _ := json.Marshal(m)
				json.Unmarshal(b, info)
				if v, ok := m["state"]; ok {
					info.Status = fmt.Sprintf("%v", v)
				}
				if v, ok := m["active_model"]; ok {
					info.Model = fmt.Sprintf("%v", v)
				}
				info.Updated = now
			}
		} else {
			var tmp interface{}
			if err := json.Unmarshal(raw, &tmp); err == nil {
				if m, ok := tmp.(map[string]interface{}); ok {
					info.FullConfig = m
					if err := json.Unmarshal(raw, info); err != nil {
						HndlErr("[MQTT] Error auto-mapping node info", err)
					}
					if v, ok := m["active_model"]; ok {
						info.Model = fmt.Sprintf("%v", v)
					}
					for _, key := range []string{"conf", "config"} {
						if cfg, ex := m[key]; ex {
							if cm, ok := cfg.(map[string]interface{}); ok {
								for k, v := range cm {
									info.FullConfig[k] = v
								}
								cfgRaw, _ := json.Marshal(cm)
								json.Unmarshal(cfgRaw, info)
							}
						}
					}
				} else {
					info.Status = fmt.Sprintf("%v", tmp)
				}
				info.Updated = now
			}
		}
	case "monitor":
		monStr := string(raw)
		info.Logs = append(info.Logs, "[MON] "+monStr)
		if len(info.Logs) > 10 {
			info.Logs = info.Logs[len(info.Logs)-10:]
		}

		if strings.HasPrefix(monStr, "M,") {
			parts := strings.Split(monStr, ",")
			if len(parts) >= 3 {
				if ram, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					info.RamFreeBytes = ram
				}
				if parts[2] == "1" || strings.ToLower(parts[2]) == "true" {
					t := true
					info.SD_OK = &t
				} else {
					f := false
					info.SD_OK = &f
				}
			}
		} else {
			var m map[string]interface{}
			if err := json.Unmarshal(raw, &m); err == nil {
				info.FullConfig = m
				if v, ok := m["ram"]; ok {
					if val, ok := v.(float64); ok {
						info.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ram_free"]; ok {
					if val, ok := v.(float64); ok {
						info.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ram_free_bytes"]; ok {
					if val, ok := v.(float64); ok {
						info.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ip"]; ok {
					info.IP = fmt.Sprintf("%v", v)
				}
				if v, ok := m["ck"]; ok {
					info.Ck = fmt.Sprintf("%v", v)
				}
				if v, ok := m["area"]; ok {
					info.Area = fmt.Sprintf("%v", v)
				}
				if v, ok := m["no"]; ok {
					info.No = fmt.Sprintf("%v", v)
				}
				if v, ok := m["no_t1"]; ok {
					if f, ok := v.(float64); ok {
						info.NoT1 = int(f)
					}
				}
				if v, ok := m["no_t2"]; ok {
					if f, ok := v.(float64); ok {
						info.NoT2 = int(f)
					}
				}
				if v, ok := m["no_t3"]; ok {
					if f, ok := v.(float64); ok {
						info.NoT3 = int(f)
					}
				}
				if v, ok := m["no_sht"]; ok {
					if f, ok := v.(float64); ok {
						info.NoSHT = int(f)
					}
				}
				if v, ok := m["no_p1"]; ok {
					if f, ok := v.(float64); ok {
						info.NoP1 = int(f)
					}
				}
				if v, ok := m["no_p2"]; ok {
					if f, ok := v.(float64); ok {
						info.NoP2 = int(f)
					}
				}
				if v, ok := m["no_p3"]; ok {
					if f, ok := v.(float64); ok {
						info.NoP3 = int(f)
					}
				}
				if v, ok := m["sd_ok"]; ok {
					if b, ok := v.(bool); ok {
						info.SD_OK = &b
					}
				}
				if v, ok := m["model"]; ok {
					info.Model = fmt.Sprintf("%v", v)
				}
				if v, ok := m["version"]; ok {
					info.Version = fmt.Sprintf("%v", v)
				}
				if v, ok := m["node_prefix"]; ok {
					info.Prefix = fmt.Sprintf("%v", v)
				}
				if data, ok := m["data"]; ok {
					if dm, ok := data.(map[string]interface{}); ok {
						if t1, ok := dm["t1"].(float64); ok {
							info.CurT1 = t1
						}
						if t2, ok := dm["t2"].(float64); ok {
							info.CurT2 = t2
						}
						if t3, ok := dm["t3"].(float64); ok {
							info.CurT3 = t3
						}
						if p1, ok := dm["p1"].(float64); ok {
							info.CurP1 = int(p1)
						}
						if p2, ok := dm["p2"].(float64); ok {
							info.CurP2 = int(p2)
						}
						if p3, ok := dm["p3"].(float64); ok {
							info.CurP3 = int(p3)
						}
						if st, ok := dm["sht_t"].(float64); ok {
							info.CurSHT_T = st
						}
						if sh, ok := dm["sht_h"].(float64); ok {
							info.CurSHT_H = sh
						}
					}
				}
				if v, ok := m["relay"]; ok {
					relayOn := false
					switch val := v.(type) {
					case bool:
						relayOn = val
					case float64:
						relayOn = val > 0
					case string:
						relayOn = val == "ON" || val == "1" || val == "true"
					}

					if relayOn != info.Relay {
						info.Relay = relayOn
						AlarmMutex.Lock()
						prev, exists := LastAlarmState[nodeID]
						if relayOn && (!exists || !prev) {
							LastAlarmState[nodeID] = true
							go sendRelayTelegram(nodeID, info, true)
						} else if !relayOn && exists && prev {
							LastAlarmState[nodeID] = false
							go sendRelayTelegram(nodeID, info, false)
						}
						AlarmMutex.Unlock()
					}
				}
			}
		}
		info.Updated = now
	case "log":
		line := string(raw)
		line = strings.TrimSpace(line)
		if line != "" {
			info.Logs = append(info.Logs, "[LOG] "+line)
			if len(info.Logs) > 10 {
				info.Logs = info.Logs[len(info.Logs)-10:]
			}
			info.Updated = now
		}
	}
	NodeStatus[nodeID] = info
}

func sendRelayTelegram(nodeID string, info *NodeInfo, isActive bool) {
	statusStr := "🚨 *ALARM ACTIVE*"
	if !isActive {
		statusStr = "✅ *ALARM CLEARED*"
	}

	emoji := "🔋"
	if isActive {
		emoji = "⚠️"
	}

	message := fmt.Sprintf("%s\n", statusStr)
	message += fmt.Sprintf("*Device:* `%s` (%s)\n", nodeID, info.Model)
	message += fmt.Sprintf("*Site:* CK %s - Area %s (Node #%s)\n", info.Ck, info.Area, info.No)
	message += "----------------------------\n"

	message += "*TEMPERATURES:*\n"
	if info.CurT1 > -100 {
		message += fmt.Sprintf("• T1: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT1, info.MinT1, info.MaxT1)
	}
	if info.CurT2 > -100 {
		message += fmt.Sprintf("• T2: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT2, info.MinT2, info.MaxT2)
	}
	if info.CurT3 > -100 {
		message += fmt.Sprintf("• T3: `%.1f°C` (Limit: %.1f - %.1f)\n", info.CurT3, info.MinT3, info.MaxT3)
	}

	if info.CurSHT_T > -40 {
		message += fmt.Sprintf("• SHT: `%.1f°C` / `%.1f%%`RH\n", info.CurSHT_T, info.CurSHT_H)
	}

	message += "\n*DOOR STATUS:*\n"
	message += fmt.Sprintf("• P1: %s\n", formatDoor(info.CurP1))
	message += fmt.Sprintf("• P2: %s\n", formatDoor(info.CurP2))
	message += fmt.Sprintf("• P3: %s\n", formatDoor(info.CurP3))

	message += "----------------------------\n"
	if isActive {
		message += fmt.Sprintf("%s *RELAY STATUS: ON*\n", emoji)
		message += "_Immediately check the storage area!_"
	} else {
		message += "🔋 *RELAY STATUS: OFF*\n"
		message += "_System normalized._"
	}

	SendTelegramMessage(message)
}

func formatDoor(val int) string {
	if val == 0 {
		return "🟢 CLOSED"
	}
	return "🔴 OPEN"
}
