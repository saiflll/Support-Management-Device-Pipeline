package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func HandleGetNodes(c *fiber.Ctx) error {
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()

	ndsLts := make(map[string]*NodeInfo)
	macKeId := make(map[string]string)

	for id, inf := range NodeStatus {
		mtc := MacRegex.FindAllString(id, -1)
		mac := id
		if len(mtc) > 0 {
			mac = mtc[len(mtc)-1]
		}

		exId, ada := macKeId[mac]
		if !ada {
			macKeId[mac] = id
		} else if inf.Updated > NodeStatus[exId].Updated {
			macKeId[mac] = id
		}
	}

	for _, ltsId := range macKeId {
		ndsLts[ltsId] = NodeStatus[ltsId]
	}

	wkt := time.Now()
	for _, inf := range ndsLts {
		if inf.Updated != "" {
			updWkt, err := time.ParseInLocation("2006-01-02 15:04:05", inf.Updated, time.Local)
			if err == nil {
				if inf.Status != "offline" && wkt.Sub(updWkt) > 45*time.Second {
					inf.Status = "offline"
				}
			}
		} else {
			inf.Status = "offline"
		}
	}

	return c.JSON(ndsLts)
}

func HandleGetModels(c *fiber.Ctx) error {
	return c.JSON(ModelRegistry)
}

func HandleGetModelByName(c *fiber.Ctx) error {
	nmMdl := c.Params("name")
	if cfg, ada := ModelRegistry[nmMdl]; ada {
		return c.JSON(cfg)
	}

	prfMdl := GetEnv("MODEL_PREFIX", "TEMP|")
	if strings.HasPrefix(nmMdl, prfMdl) {
		if cfg, ada := ModelRegistry["TEMP"]; ada {
			return c.JSON(cfg)
		}
	}

	if cfg, ada := ModelRegistry["GENERIC"]; ada {
		return c.JSON(cfg)
	}

	return c.Status(404).JSON(fiber.Map{"error": "model not found"})
}

func HandleGetNodeConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	b := []byte("cmd=get_config")
	MqttClient.Publish(fmt.Sprintf("nodes/%s/command", id), 0, false, b)

	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	if inf, ok := NodeStatus[id]; ok {
		return c.JSON(inf.FullConfig)
	}
	return c.Status(404).JSON(fiber.Map{"error": "node not found"})
}

func HandleDeleteNode(c *fiber.Ctx) error {
	id := c.Params("id")
	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	mtc := MacRegex.FindAllString(id, -1)
	if len(mtc) == 0 {
		if _, ok := NodeStatus[id]; ok {
			delete(NodeStatus, id)
			MqttClient.Publish(fmt.Sprintf("nodes/%s/status", id), 0, true, []byte{})
			MqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", id), 0, true, []byte{})
			return c.JSON(fiber.Map{"status": "deleted", "node": id})
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "node not found"})
	}
	tgtMac := mtc[len(mtc)-1]

	ndsHps := []string{}
	jmlHps := 0
	for ndId := range NodeStatus {
		macMtc := MacRegex.FindAllString(ndId, -1)
		if len(macMtc) > 0 && macMtc[len(macMtc)-1] == tgtMac {
			ndsHps = append(ndsHps, ndId)
		}
	}

	for _, ndId := range ndsHps {
		delete(NodeStatus, ndId)
		MqttClient.Publish(fmt.Sprintf("nodes/%s/status", ndId), 0, true, []byte{})
		MqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", ndId), 0, true, []byte{})
		jmlHps++
	}

	if jmlHps > 0 {
		return c.JSON(fiber.Map{"status": "deleted", "mac": tgtMac, "count": jmlHps})
	}

	return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no nodes found for the given ID or MAC"})
}

func HandleGetFiles(c *fiber.Ctx) error {
	FileMutex.Lock()
	defer FileMutex.Unlock()

	tgtDr := filepath.Join("static", "uploads")
	ent, err := os.ReadDir(tgtDr)
	if err == nil {
		flDsk := make(map[string]bool)
		for _, e := range ent {
			if !e.IsDir() {
				flDsk[e.Name()] = true
			}
		}

		tmpMap := make(map[string]FileInfo)
		for nm, inf := range FileInfos {
			if flDsk[nm] {
				tmpMap[nm] = inf
			} else {
				Lg("[SYNC] Removing stale entry: %s", nm)
			}
		}
		FileInfos = tmpMap

		for nm := range flDsk {
			if _, ada := FileInfos[nm]; !ada {
				inf, err := os.Stat(filepath.Join(tgtDr, nm))
				if err == nil {
					Lg("[SYNC] Adding missing disk file: %s", nm)
					FileInfos[nm] = FileInfo{
						Name:       nm,
						URL:        "/files/" + nm,
						UploadTime: inf.ModTime(),
						Size:       inf.Size(),
					}
				}
			}
		}
	} else {
		HndlErr("SYNC Warning: could not read upload directory", err)
	}

	fls := make([]FileInfo, 0, len(FileInfos))
	for _, f := range FileInfos {
		fls = append(fls, f)
	}
	return c.JSON(fls)
}

func HandleDeleteFile(c *fiber.Ctx) error {
	nm := c.Params("*")
	if nm == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename required"})
	}
	cln := filepath.Base(nm)
	pth := filepath.Join("static", "uploads", cln)

	FileMutex.Lock()
	defer FileMutex.Unlock()

	if _, err := os.Stat(pth); os.IsNotExist(err) {
		delete(FileInfos, cln)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found on disk, registry cleaned"})
	}
	if err := os.Remove(pth); err != nil {
		HndlErr(fmt.Sprintf("Failed to delete file %s", pth), err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed delete"})
	}
	delete(FileInfos, cln)
	return c.JSON(fiber.Map{"status": "deleted", "name": cln})
}

func HandleRenameFile(c *fiber.Ctx) error {
	nm := c.Params("name")
	type RenameRequest struct {
		NewName string `json:"new_name"`
	}
	var req RenameRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	nmBaru := filepath.Base(req.NewName)
	if nmBaru == "" || nmBaru == "." || nmBaru == ".." {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid new name"})
	}

	FileMutex.Lock()
	defer FileMutex.Unlock()

	if _, ok := FileInfos[nm]; !ok {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}

	pthLma := filepath.Join("static", "uploads", nm)
	pthBru := filepath.Join("static", "uploads", nmBaru)

	if err := os.Rename(pthLma, pthBru); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rename file"})
	}

	inf := FileInfos[nm]
	delete(FileInfos, nm)
	inf.Name = nmBaru
	inf.URL = "/files/" + nmBaru
	FileInfos[nmBaru] = inf

	return c.JSON(inf)
}

func HandleUpload(c *fiber.Ctx) error {
	frm, err := c.MultipartForm()
	if err != nil {
		HndlErr("Upload failed", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "multipart form required"})
	}

	fls := frm.File["file"]
	if len(fls) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "no files provided"})
	}

	var flsUgg []string
	FileMutex.Lock()
	defer FileMutex.Unlock()

	for _, f := range fls {
		nmBse := filepath.Base(f.Filename)
		dst := filepath.Join("static", "uploads", nmBse)

		if err := c.SaveFile(f, dst); err != nil {
			HndlErr(fmt.Sprintf("Failed to save file %s", dst), err)
			continue
		}

		FileInfos[nmBse] = FileInfo{
			Name:       nmBse,
			URL:        "/files/" + nmBse,
			UploadTime: time.Now(),
			Size:       f.Size,
		}
		flsUgg = append(flsUgg, nmBse)
		Lg("Successfully uploaded: %s as %s (%d bytes)", f.Filename, nmBse, f.Size)
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"count":  len(flsUgg),
		"files":  flsUgg,
	})
}

func HandlePostConfig(c *fiber.Ctx) error {
	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	ndId, ok := req["node"].(string)
	if !ok || ndId == "" {
		return c.Status(400).JSON(fiber.Map{"error": "node ID required"})
	}

	psn := make(map[string]interface{})
	for k, v := range req {
		if k == "node" {
			continue
		}
		psn[k] = v
	}

	if _, ada := psn["cmd"]; !ada {
		psn["cmd"] = "set_config"
	}

	NodeMutex.Lock()
	if inf, ok := NodeStatus[ndId]; ok {
		if inf.FullConfig == nil {
			inf.FullConfig = make(map[string]interface{})
		}
		for k, v := range psn {
			if k == "cmd" {
				continue
			}
			inf.FullConfig[k] = v
		}
	}
	NodeMutex.Unlock()

	val := url.Values{}
	for k, v := range psn {
		val.Set(k, fmt.Sprintf("%v", v))
	}
	b := []byte(val.Encode())

	tpc := fmt.Sprintf("nodes/%s/command", ndId)
	tkn := MqttClient.Publish(tpc, 0, false, b)
	tkn.Wait()

	return c.JSON(fiber.Map{
		"status": "ok",
		"topic":  tpc,
		"cmd":    psn["cmd"],
	})
}

func HandlePostOTA(c *fiber.Ctx) error {
	type O struct {
		Node string `json:"node"`
		URL  string `json:"url"`
	}
	var o O
	if err := c.BodyParser(&o); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	b := []byte(fmt.Sprintf("cmd=ota&url=%s", url.QueryEscape(o.URL)))
	tpc := fmt.Sprintf("nodes/%s/command", o.Node)
	tkn := MqttClient.Publish(tpc, 0, false, b)
	tkn.Wait()
	return c.JSON(fiber.Map{"status": "OTA triggered", "topic": tpc})
}

func HandlePostReboot(c *fiber.Ctx) error {
	type R struct {
		Node string `json:"node"`
	}
	var r R
	if err := c.BodyParser(&r); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	b := []byte("cmd=reboot")
	tpc := fmt.Sprintf("nodes/%s/command", r.Node)
	tkn := MqttClient.Publish(tpc, 0, false, b)
	tkn.Wait()
	return c.JSON(fiber.Map{"status": "Reboot triggered", "topic": tpc})
}

func HandleGetLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	if inf, ok := NodeStatus[id]; ok {
		return c.JSON(fiber.Map{"node": id, "logs": inf.Logs})
	}
	return c.Status(404).JSON(fiber.Map{"error": "node not found"})
}

func HandleGetForwarderStatus(c *fiber.Ctx) error {
	urlStr := GetEnv("FORWARDER_URL", "http://forwarder:8888/forwarder/status")
	res, err := http.Get(urlStr)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer res.Body.Close()
	dt, _ := io.ReadAll(res.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(dt)
}

func HandleGetMonitorStatus(c *fiber.Ctx) error {
	urlStr := GetEnv("MONITOR_URL", "http://monitor:9090/status")
	res, err := http.Get(urlStr)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Monitor service tidak tersedia"})
	}
	defer res.Body.Close()
	dt, _ := io.ReadAll(res.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(dt)
}

func HandleGetPipelines(c *fiber.Ctx) error {
	urlStr := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	res, err := http.Get(urlStr + "/pipelines")
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer res.Body.Close()
	dt, _ := io.ReadAll(res.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(dt)
}

func HandlePostPipelines(c *fiber.Ctx) error {
	urlStr := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	res, err := http.Post(urlStr+"/pipelines", "application/json", strings.NewReader(string(c.Body())))
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer res.Body.Close()
	dt, _ := io.ReadAll(res.Body)
	c.Status(res.StatusCode).Set("Content-Type", "application/json")
	return c.Send(dt)
}

func HandleDeletePipeline(c *fiber.Ctx) error {
	id := c.Params("id")
	urlStr := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	req, _ := http.NewRequest("DELETE", urlStr+"/pipelines/"+id, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer res.Body.Close()
	return c.SendStatus(res.StatusCode)
}
