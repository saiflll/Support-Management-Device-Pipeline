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

	var ndId, sub string
	if prts[0] == "nodes" && len(prts) >= 3 {
		ndId = prts[1]
		sub = prts[2]
	} else if len(prts) >= 2 {
		ndId = prts[0]
		sub = prts[1]
	} else {
		return
	}

	raw := psn.Payload()
	wkt := time.Now().Format("2006-01-02 15:04:05")

	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	if _, ok := NodeStatus[ndId]; !ok {
		mtcMac := MacRegex.FindAllString(ndId, -1)
		if len(mtcMac) > 0 {
			mac := mtcMac[len(mtcMac)-1]
			for oldId, oldInf := range NodeStatus {
				if oldId == ndId {
					continue
				}
				oldMacMtc := MacRegex.FindAllString(oldId, -1)
				if len(oldMacMtc) > 0 && oldMacMtc[len(oldMacMtc)-1] == mac {
					Lg("MAC match! Migrating '%s' -> '%s'", oldId, ndId)
					newInf := &NodeInfo{
						Ck:           oldInf.Ck,
						Area:         oldInf.Area,
						No:           oldInf.No,
						MinT1:        oldInf.MinT1,
						MaxT1:        oldInf.MaxT1,
						MinT2:        oldInf.MinT2,
						MaxT2:        oldInf.MaxT2,
						MinT3:        oldInf.MinT3,
						MaxT3:        oldInf.MaxT3,
						SHTSuhuMin:   oldInf.SHTSuhuMin,
						SHTSuhuMax:   oldInf.SHTSuhuMax,
						SHTHumMin:    oldInf.SHTHumMin,
						SHTHumMax:    oldInf.SHTHumMax,
						Interval:     oldInf.Interval,
						Prefix:       oldInf.Prefix,
						Model:        oldInf.Model,
						Version:      oldInf.Version,
						AppMode:      oldInf.AppMode,
						Trans:        oldInf.Trans,
						PassCode:     oldInf.PassCode,
						IP:           oldInf.IP,
						RamFreeBytes: oldInf.RamFreeBytes,
						SD_OK:        oldInf.SD_OK,
						Logs:         oldInf.Logs,
						Status:       "online",
						Updated:      wkt,
					}
					NodeStatus[ndId] = newInf
					delete(NodeStatus, oldId)
					break
				}
			}
		}
	}

	if _, ok := NodeStatus[ndId]; !ok {
		if len(NodeStatus) >= 1000 {
			Lg("[MQTT] Node limit reached, dropping %s", ndId)
			return
		}
		NodeStatus[ndId] = &NodeInfo{}
	}
	inf := NodeStatus[ndId]

	switch sub {
	case "status":
		str := string(raw)
		if strings.HasPrefix(str, "state=") || strings.HasPrefix(str, "cmd=") {
			val, err := url.ParseQuery(str)
			if err != nil {
				HndlErr(fmt.Sprintf("[MQTT] Warning: URL decode failed for %s, parsing anyway", ndId), err)
			}
			fldBool := map[string]bool{"alarm_enabled": true, "sd_ok": true, "relay": true}
			if val != nil {
				m := make(map[string]interface{})
				for k, v := range val {
					if len(v) > 0 {
						if fldBool[k] {
							m[k] = v[0] == "true" || v[0] == "1"
						} else if f, err := strconv.ParseFloat(v[0], 64); err == nil {
							m[k] = f
						} else {
							m[k] = v[0]
						}
					}
				}
				inf.FullConfig = m
				b, _ := json.Marshal(m)
				json.Unmarshal(b, inf)
				if v, ok := m["state"]; ok {
					inf.Status = fmt.Sprintf("%v", v)
				}
				if v, ok := m["active_model"]; ok {
					inf.Model = fmt.Sprintf("%v", v)
				}
				inf.Updated = wkt
			}
		} else {
			var tmp interface{}
			if err := json.Unmarshal(raw, &tmp); err == nil {
				if m, ok := tmp.(map[string]interface{}); ok {
					inf.FullConfig = m
					if err := json.Unmarshal(raw, inf); err != nil {
						HndlErr("[MQTT] Error auto-mapping node info", err)
					}
					if v, ok := m["active_model"]; ok {
						inf.Model = fmt.Sprintf("%v", v)
					}
					for _, key := range []string{"conf", "config"} {
						if cfg, ex := m[key]; ex {
							if cm, ok := cfg.(map[string]interface{}); ok {
								for k, v := range cm {
									inf.FullConfig[k] = v
								}
								cfgRaw, _ := json.Marshal(cm)
								json.Unmarshal(cfgRaw, inf)
							}
						}
					}
				} else {
					inf.Status = fmt.Sprintf("%v", tmp)
				}
				inf.Updated = wkt
			}
		}
	case "monitor":
		str := string(raw)
		inf.Logs = append(inf.Logs, "[MON] "+str)
		if len(inf.Logs) > 10 {
			inf.Logs = inf.Logs[len(inf.Logs)-10:]
		}

		if strings.HasPrefix(str, "M,") {
			parts := strings.Split(str, ",")
			if len(parts) >= 3 {
				if ram, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					inf.RamFreeBytes = ram
				}
				if parts[2] == "1" || strings.ToLower(parts[2]) == "true" {
					t := true
					inf.SD_OK = &t
				} else {
					f := false
					inf.SD_OK = &f
				}
			}
		} else {
			var m map[string]interface{}
			if err := json.Unmarshal(raw, &m); err == nil {
				inf.FullConfig = m
				if v, ok := m["ram"]; ok {
					if val, ok := v.(float64); ok {
						inf.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ram_free"]; ok {
					if val, ok := v.(float64); ok {
						inf.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ram_free_bytes"]; ok {
					if val, ok := v.(float64); ok {
						inf.RamFreeBytes = int64(val)
					}
				}
				if v, ok := m["ip"]; ok {
					inf.IP = fmt.Sprintf("%v", v)
				}
				if v, ok := m["ck"]; ok {
					inf.Ck = FlexString(fmt.Sprintf("%v", v))
				}
				if v, ok := m["area"]; ok {
					inf.Area = FlexString(fmt.Sprintf("%v", v))
				}
				if v, ok := m["no"]; ok {
					inf.No = FlexString(fmt.Sprintf("%v", v))
				}
				if v, ok := m["no_t1"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoT1 = int(f)
					}
				}
				if v, ok := m["no_t2"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoT2 = int(f)
					}
				}
				if v, ok := m["no_t3"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoT3 = int(f)
					}
				}
				if v, ok := m["no_sht"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoSHT = int(f)
					}
				}
				if v, ok := m["no_p1"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoP1 = int(f)
					}
				}
				if v, ok := m["no_p2"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoP2 = int(f)
					}
				}
				if v, ok := m["no_p3"]; ok {
					if f, ok := v.(float64); ok {
						inf.NoP3 = int(f)
					}
				}
				if v, ok := m["sd_ok"]; ok {
					if b, ok := v.(bool); ok {
						inf.SD_OK = &b
					}
				}
				if v, ok := m["model"]; ok {
					inf.Model = fmt.Sprintf("%v", v)
				}
				if v, ok := m["version"]; ok {
					inf.Version = fmt.Sprintf("%v", v)
				}
				if v, ok := m["node_prefix"]; ok {
					inf.Prefix = fmt.Sprintf("%v", v)
				}
				if data, ok := m["data"]; ok {
					if dm, ok := data.(map[string]interface{}); ok {
						if t1, ok := dm["t1"].(float64); ok {
							inf.CurT1 = t1
						}
						if t2, ok := dm["t2"].(float64); ok {
							inf.CurT2 = t2
						}
						if t3, ok := dm["t3"].(float64); ok {
							inf.CurT3 = t3
						}
						if p1, ok := dm["p1"].(float64); ok {
							inf.CurP1 = int(p1)
						}
						if p2, ok := dm["p2"].(float64); ok {
							inf.CurP2 = int(p2)
						}
						if p3, ok := dm["p3"].(float64); ok {
							inf.CurP3 = int(p3)
						}
						if st, ok := dm["sht_t"].(float64); ok {
							inf.CurSHT_T = st
						}
						if sh, ok := dm["sht_h"].(float64); ok {
							inf.CurSHT_H = sh
						}
					}
				}
				if v, ok := m["relay"]; ok {
					isRly := false
					switch val := v.(type) {
					case bool:
						isRly = val
					case float64:
						isRly = val > 0
					case string:
						isRly = val == "ON" || val == "1" || val == "true"
					}

					if isRly != inf.Relay {
						inf.Relay = isRly
						AlarmMutex.Lock()
						prev, ada := LastAlarmState[ndId]
						if isRly && (!ada || !prev) {
							LastAlarmState[ndId] = true
							go sendRelayTelegram(ndId, inf, true)
						} else if !isRly && ada && prev {
							LastAlarmState[ndId] = false
							go sendRelayTelegram(ndId, inf, false)
						}
						AlarmMutex.Unlock()
					}
				}
			}
		}
		inf.Updated = wkt
	case "log":
		ln := string(raw)
		ln = strings.TrimSpace(ln)
		if ln != "" {
			inf.Logs = append(inf.Logs, "[LOG] "+ln)
			if len(inf.Logs) > 10 {
				inf.Logs = inf.Logs[len(inf.Logs)-10:]
			}
			inf.Updated = wkt
		}
	}
	NodeStatus[ndId] = inf
}

func sendRelayTelegram(ndId string, inf *NodeInfo, aktf bool) {
	stsStr := "🚨 *ALARM ACTIVE*"
	if !aktf {
		stsStr = "✅ *ALARM CLEARED*"
	}

	emj := "🔋"
	if aktf {
		emj = "⚠️"
	}

	psn := fmt.Sprintf("%s\n", stsStr)
	psn += fmt.Sprintf("*Device:* `%s` (%s)\n", ndId, inf.Model)
	psn += fmt.Sprintf("*Site:* CK %s - Area %s (Node #%s)\n", inf.Ck, inf.Area, inf.No)
	psn += "----------------------------\n"

	psn += "*TEMPERATURES:*\n"
	if inf.CurT1 > -100 {
		psn += fmt.Sprintf("• T1: `%.1f°C` (Limit: %.1f - %.1f)\n", inf.CurT1, inf.MinT1, inf.MaxT1)
	}
	if inf.CurT2 > -100 {
		psn += fmt.Sprintf("• T2: `%.1f°C` (Limit: %.1f - %.1f)\n", inf.CurT2, inf.MinT2, inf.MaxT2)
	}
	if inf.CurT3 > -100 {
		psn += fmt.Sprintf("• T3: `%.1f°C` (Limit: %.1f - %.1f)\n", inf.CurT3, inf.MinT3, inf.MaxT3)
	}

	if inf.CurSHT_T > -40 {
		psn += fmt.Sprintf("• SHT: `%.1f°C` / `%.1f%%`RH\n", inf.CurSHT_T, inf.CurSHT_H)
	}

	psn += "\n*DOOR STATUS:*\n"
	psn += fmt.Sprintf("• P1: %s\n", formatDoor(inf.CurP1))
	psn += fmt.Sprintf("• P2: %s\n", formatDoor(inf.CurP2))
	psn += fmt.Sprintf("• P3: %s\n", formatDoor(inf.CurP3))

	psn += "----------------------------\n"
	if aktf {
		psn += fmt.Sprintf("%s *RELAY STATUS: ON*\n", emj)
		psn += "_Immediately check the storage area!_"
	} else {
		psn += "🔋 *RELAY STATUS: OFF*\n"
		psn += "_System normalized._"
	}

	SendTelegramMessage(psn)
}

func formatDoor(val int) string {
	if val == 0 {
		return "🟢 CLOSED"
	}
	return "🔴 OPEN"
}
