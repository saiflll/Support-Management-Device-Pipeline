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

	latestNodes := make(map[string]*NodeInfo)
	macToNodeID := make(map[string]string)

	for id, info := range NodeStatus {
		matches := MacRegex.FindAllString(id, -1)
		mac := id
		if len(matches) > 0 {
			mac = matches[len(matches)-1]
		}

		existingNodeID, found := macToNodeID[mac]
		if !found {
			macToNodeID[mac] = id
		} else if info.Updated > NodeStatus[existingNodeID].Updated {
			macToNodeID[mac] = id
		}
	}

	for _, latestID := range macToNodeID {
		latestNodes[latestID] = NodeStatus[latestID]
	}

	now := time.Now()
	for _, info := range latestNodes {
		if info.Updated != "" {
			updatedTime, err := time.ParseInLocation("2006-01-02 15:04:05", info.Updated, time.Local)
			if err == nil {
				if info.Status != "offline" && now.Sub(updatedTime) > 45*time.Second {
					info.Status = "offline"
				}
			}
		} else {
			info.Status = "offline"
		}
	}

	return c.JSON(latestNodes)
}

func HandleGetModels(c *fiber.Ctx) error {
	return c.JSON(ModelRegistry)
}

func HandleGetModelByName(c *fiber.Ctx) error {
	modelName := c.Params("name")
	if config, exists := ModelRegistry[modelName]; exists {
		return c.JSON(config)
	}

	modelPrefix := GetEnv("MODEL_PREFIX", "TEMP|")
	if strings.HasPrefix(modelName, modelPrefix) {
		if config, exists := ModelRegistry["TEMP"]; exists {
			return c.JSON(config)
		}
	}

	if config, exists := ModelRegistry["GENERIC"]; exists {
		return c.JSON(config)
	}

	return c.Status(404).JSON(fiber.Map{"error": "model not found"})
}

func HandleGetNodeConfig(c *fiber.Ctx) error {
	id := c.Params("id")
	b := []byte("cmd=get_config")
	MqttClient.Publish(fmt.Sprintf("nodes/%s/command", id), 0, false, b)

	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	if info, ok := NodeStatus[id]; ok {
		return c.JSON(info.FullConfig)
	}
	return c.Status(404).JSON(fiber.Map{"error": "node not found"})
}

func HandleDeleteNode(c *fiber.Ctx) error {
	id := c.Params("id")
	NodeMutex.Lock()
	defer NodeMutex.Unlock()

	matches := MacRegex.FindAllString(id, -1)
	if len(matches) == 0 {
		if _, ok := NodeStatus[id]; ok {
			delete(NodeStatus, id)
			MqttClient.Publish(fmt.Sprintf("nodes/%s/status", id), 0, true, []byte{})
			MqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", id), 0, true, []byte{})
			return c.JSON(fiber.Map{"status": "deleted", "node": id})
		}
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "node not found"})
	}
	targetMac := matches[len(matches)-1]

	nodesToDelete := []string{}
	deletedCount := 0
	for nodeID := range NodeStatus {
		nodeMacMatches := MacRegex.FindAllString(nodeID, -1)
		if len(nodeMacMatches) > 0 && nodeMacMatches[len(nodeMacMatches)-1] == targetMac {
			nodesToDelete = append(nodesToDelete, nodeID)
		}
	}

	for _, nodeID := range nodesToDelete {
		delete(NodeStatus, nodeID)
		MqttClient.Publish(fmt.Sprintf("nodes/%s/status", nodeID), 0, true, []byte{})
		MqttClient.Publish(fmt.Sprintf("nodes/%s/monitor", nodeID), 0, true, []byte{})
		deletedCount++
	}

	if deletedCount > 0 {
		return c.JSON(fiber.Map{"status": "deleted", "mac": targetMac, "count": deletedCount})
	}

	return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no nodes found for the given ID or MAC"})
}

func HandleGetFiles(c *fiber.Ctx) error {
	FileMutex.Lock()
	defer FileMutex.Unlock()

	targetDir := filepath.Join("static", "uploads")
	entries, err := os.ReadDir(targetDir)
	if err == nil {
		diskFiles := make(map[string]bool)
		for _, entry := range entries {
			if !entry.IsDir() {
				diskFiles[entry.Name()] = true
			}
		}

		tempMap := make(map[string]FileInfo)
		for name, info := range FileInfos {
			if diskFiles[name] {
				tempMap[name] = info
			} else {
				Lg("[SYNC] Removing stale entry: %s", name)
			}
		}
		FileInfos = tempMap

		for name := range diskFiles {
			if _, exists := FileInfos[name]; !exists {
				info, err := os.Stat(filepath.Join(targetDir, name))
				if err == nil {
					Lg("[SYNC] Adding missing disk file: %s", name)
					FileInfos[name] = FileInfo{
						Name:       name,
						URL:        "/files/" + name,
						UploadTime: info.ModTime(),
						Size:       info.Size(),
					}
				}
			}
		}
	} else {
		HndlErr("SYNC Warning: could not read upload directory", err)
	}

	files := make([]FileInfo, 0, len(FileInfos))
	for _, f := range FileInfos {
		files = append(files, f)
	}
	return c.JSON(files)
}

func HandleDeleteFile(c *fiber.Ctx) error {
	name := c.Params("*")
	if name == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "filename required"})
	}
	clean := filepath.Base(name)
	path := filepath.Join("static", "uploads", clean)

	FileMutex.Lock()
	defer FileMutex.Unlock()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		delete(FileInfos, clean)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found on disk, registry cleaned"})
	}
	if err := os.Remove(path); err != nil {
		HndlErr(fmt.Sprintf("Failed to delete file %s", path), err)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed delete"})
	}
	delete(FileInfos, clean)
	return c.JSON(fiber.Map{"status": "deleted", "name": clean})
}

func HandleRenameFile(c *fiber.Ctx) error {
	name := c.Params("name")
	type RenameRequest struct {
		NewName string `json:"new_name"`
	}
	var req RenameRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	cleanNewName := filepath.Base(req.NewName)
	if cleanNewName == "" || cleanNewName == "." || cleanNewName == ".." {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid new name"})
	}

	FileMutex.Lock()
	defer FileMutex.Unlock()

	if _, ok := FileInfos[name]; !ok {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}

	oldPath := filepath.Join("static", "uploads", name)
	newPath := filepath.Join("static", "uploads", cleanNewName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rename file"})
	}

	fileInfo := FileInfos[name]
	delete(FileInfos, name)
	fileInfo.Name = cleanNewName
	fileInfo.URL = "/files/" + cleanNewName
	FileInfos[cleanNewName] = fileInfo

	return c.JSON(fileInfo)
}

func HandleUpload(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		HndlErr("Upload failed", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "multipart form required"})
	}

	files := form.File["file"]
	if len(files) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "no files provided"})
	}

	var uploadedFiles []string
	FileMutex.Lock()
	defer FileMutex.Unlock()

	for _, f := range files {
		baseName := filepath.Base(f.Filename)
		dst := filepath.Join("static", "uploads", baseName)

		if err := c.SaveFile(f, dst); err != nil {
			HndlErr(fmt.Sprintf("Failed to save file %s", dst), err)
			continue
		}

		FileInfos[baseName] = FileInfo{
			Name:       baseName,
			URL:        "/files/" + baseName,
			UploadTime: time.Now(),
			Size:       f.Size,
		}
		uploadedFiles = append(uploadedFiles, baseName)
		Lg("Successfully uploaded: %s as %s (%d bytes)", f.Filename, baseName, f.Size)
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"count":  len(uploadedFiles),
		"files":  uploadedFiles,
	})
}

func HandlePostConfig(c *fiber.Ctx) error {
	var req map[string]interface{}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	nodeID, ok := req["node"].(string)
	if !ok || nodeID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "node ID required"})
	}

	payload := make(map[string]interface{})
	for k, v := range req {
		if k == "node" {
			continue
		}
		payload[k] = v
	}

	if _, exists := payload["cmd"]; !exists {
		payload["cmd"] = "set_config"
	}

	NodeMutex.Lock()
	if info, ok := NodeStatus[nodeID]; ok {
		if info.FullConfig == nil {
			info.FullConfig = make(map[string]interface{})
		}
		for k, v := range payload {
			if k == "cmd" {
				continue
			}
			info.FullConfig[k] = v
		}
	}
	NodeMutex.Unlock()

	values := url.Values{}
	for k, v := range payload {
		values.Set(k, fmt.Sprintf("%v", v))
	}
	b := []byte(values.Encode())

	topic := fmt.Sprintf("nodes/%s/command", nodeID)
	token := MqttClient.Publish(topic, 0, false, b)
	token.Wait()

	return c.JSON(fiber.Map{
		"status": "ok",
		"topic":  topic,
		"cmd":    payload["cmd"],
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
	topic := fmt.Sprintf("nodes/%s/command", o.Node)
	token := MqttClient.Publish(topic, 0, false, b)
	token.Wait()
	return c.JSON(fiber.Map{"status": "OTA triggered", "topic": topic})
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
	topic := fmt.Sprintf("nodes/%s/command", r.Node)
	token := MqttClient.Publish(topic, 0, false, b)
	token.Wait()
	return c.JSON(fiber.Map{"status": "Reboot triggered", "topic": topic})
}

func HandleGetLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	NodeMutex.RLock()
	defer NodeMutex.RUnlock()
	if info, ok := NodeStatus[id]; ok {
		return c.JSON(fiber.Map{"node": id, "logs": info.Logs})
	}
	return c.Status(404).JSON(fiber.Map{"error": "node not found"})
}

func HandleGetForwarderStatus(c *fiber.Ctx) error {
	forwarderURL := GetEnv("FORWARDER_URL", "http://forwarder:8888/forwarder/status")
	resp, err := http.Get(forwarderURL)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func HandleGetMonitorStatus(c *fiber.Ctx) error {
	monitorURL := GetEnv("MONITOR_URL", "http://monitor:9090/status")
	resp, err := http.Get(monitorURL)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Monitor service tidak tersedia"})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func HandleGetPipelines(c *fiber.Ctx) error {
	forwarderBase := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	resp, err := http.Get(forwarderBase + "/pipelines")
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Set("Content-Type", "application/json")
	return c.Send(body)
}

func HandlePostPipelines(c *fiber.Ctx) error {
	forwarderBase := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	resp, err := http.Post(forwarderBase+"/pipelines", "application/json", strings.NewReader(string(c.Body())))
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Status(resp.StatusCode).Set("Content-Type", "application/json")
	return c.Send(body)
}

func HandleDeletePipeline(c *fiber.Ctx) error {
	id := c.Params("id")
	forwarderBase := GetEnv("FORWARDER_API_URL", "http://forwarder:8888/api")
	req, _ := http.NewRequest("DELETE", forwarderBase+"/pipelines/"+id, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Forwarder service tidak tersedia"})
	}
	defer resp.Body.Close()
	return c.SendStatus(resp.StatusCode)
}
