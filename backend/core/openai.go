package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"evo-image-api/models"
	"evo-image-api/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	reAsset = regexp.MustCompile(`(file-service|sediment)://[A-Za-z0-9_-]+`)
)

type OpenAIClient struct {
	Account *models.Account
	Client  *http.Client
	Logger  io.Writer
}

func (c *OpenAIClient) log(format string, a ...interface{}) {
	if c.Logger != nil {
		fmt.Fprintf(c.Logger, format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

func NewOpenAIClient(acc *models.Account) (*OpenAIClient, error) {
	client, err := utils.NewTLSClient(acc.Proxy)
	if err != nil {
		return nil, err
	}
	return &OpenAIClient{
		Account: acc,
		Client:  client,
	}, nil
}

func (c *OpenAIClient) setCommonHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Account.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	if c.Account.AccountID != "" { req.Header.Set("oai-device-id", c.Account.AccountID) }
	if c.Account.SessionID != "" { req.Header.Set("Cookie", "authsession="+c.Account.SessionID) }
}

func (c *OpenAIClient) GetChatRequirements() (string, string, error) {
	traceID := utils.GenerateUUID()
	prep, err := c.ChatRequirementsPrepare(traceID)
	if err != nil { return c.GetChatRequirementsSingleStep(traceID) }

	proof := ""
	if prep.Proofofwork.Required {
		proof = utils.SolveProofToken(prep.Proofofwork.Seed, prep.Proofofwork.Difficulty, "")
	}

	if prep.Turnstile.Required { return c.GetChatRequirementsSingleStep(traceID) }

	token, _, err := c.ChatRequirementsFinalize(traceID, prep.PrepareToken, proof)
	if err != nil { return c.GetChatRequirementsSingleStep(traceID) }

	return token, proof, nil
}

type ChatRequirementsPrep struct {
	PrepareToken string `json:"prepare_token"`
	Persona      string `json:"persona"`
	Proofofwork  struct {
		Required   bool   `json:"required"`
		Seed       string `json:"seed"`
		Difficulty string `json:"difficulty"`
	} `json:"proofofwork"`
	Turnstile struct {
		Required bool `json:"required"`
	} `json:"turnstile"`
}

func (c *OpenAIClient) ChatRequirementsPrepare(traceID string) (*ChatRequirementsPrep, error) {
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/sentinel/chat-requirements/prepare", nil)
	c.setCommonHeaders(req)
	req.Header.Set("X-Oai-Turn-Trace-Id", traceID)
	resp, err := c.Client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return nil, fmt.Errorf("prep fail: %d", resp.StatusCode) }
	var out ChatRequirementsPrep
	json.NewDecoder(resp.Body).Decode(&out)
	return &out, nil
}

func (c *OpenAIClient) ChatRequirementsFinalize(traceID, prepToken, proof string) (string, string, error) {
	payload := map[string]interface{}{"prepare_token": prepToken, "proofofwork": proof}
	jb, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/sentinel/chat-requirements/finalize", bytes.NewBuffer(jb))
	c.setCommonHeaders(req)
	req.Header.Set("X-Oai-Turn-Trace-Id", traceID)
	resp, err := c.Client.Do(req)
	if err != nil { return "", "", err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return "", "", fmt.Errorf("fin fail: %d", resp.StatusCode) }
	var out struct { Token, Persona string }
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Token, out.Persona, nil
}

func (c *OpenAIClient) GetChatRequirementsSingleStep(traceID string) (string, string, error) {
	reqToken := utils.NewPOWConfig("").RequirementsToken()
	body, _ := json.Marshal(map[string]string{"p": reqToken})
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/sentinel/chat-requirements", bytes.NewBuffer(body))
	c.setCommonHeaders(req)
	req.Header.Set("X-Oai-Turn-Trace-Id", traceID)
	resp, err := c.Client.Do(req)
	if err != nil { return "", "", err }
	defer resp.Body.Close()
	var out struct {
		Token string `json:"token"`
		Proofofwork struct { Required bool; Seed, Difficulty string } `json:"proofofwork"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	proof := ""
	if out.Proofofwork.Required {
		proof = utils.SolveProofToken(out.Proofofwork.Seed, out.Proofofwork.Difficulty, "")
	}
	return out.Token, proof, nil
}

func (c *OpenAIClient) PrepareFConversation(prompt, chatToken, traceID string) (string, error) {
	payload := map[string]interface{}{
		"action": "next", "client_prepare_state": "success", "system_hints": []string{"picture_v2"},
		"partial_query": map[string]interface{}{
			"id": utils.GenerateUUID(), "author": map[string]string{"role": "user"},
			"content": map[string]interface{}{"content_type": "text", "parts": []string{prompt}},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/f/conversation/prepare", bytes.NewBuffer(body))
	c.setCommonHeaders(req)
	req.Header.Set("openai-sentinel-chat-requirements-token", chatToken)
	req.Header.Set("X-Oai-Turn-Trace-Id", traceID)
	resp, err := c.Client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	var out struct { ConduitToken string `json:"conduit_token"` }
	json.NewDecoder(resp.Body).Decode(&out)
	return out.ConduitToken, nil
}

// GenerateImage 简化版：由于 IMG2 已全量推送，不再需要多轮辩证
func (c *OpenAIClient) GenerateImage(prompt string) ([]string, error) {
	c.log("\n🚀 Generating Image (Standard IMG2 Pipeline)... Account: %s\n", c.Account.Email)
	
	traceID := utils.GenerateUUID()
	chatToken, proof, err := c.GetChatRequirements()
	if err != nil { return nil, fmt.Errorf("sentinel: %v", err) }
	
	conduitToken, _ := c.PrepareFConversation(prompt, chatToken, traceID)

	reqBody := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{{
			"id":      utils.GenerateUUID(),
			"author":  map[string]string{"role": "user"},
			"content": map[string]interface{}{"content_type": "text", "parts": []string{prompt}},
			"metadata": map[string]interface{}{"system_hints": []string{"picture_v2"}},
		}},
		"parent_message_id":     utils.GenerateUUID(),
		"model":                 "auto", // 默认自动路由到最新 gpt-5-3 / IMG2
		"client_prepare_state":  "sent",
		"system_hints":          []string{"picture_v2"},
		"history_and_training_disabled": false,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/f/conversation", bytes.NewBuffer(jsonData))
	c.setCommonHeaders(req)
	req.Header.Set("openai-sentinel-chat-requirements-token", chatToken)
	req.Header.Set("X-Oai-Turn-Trace-Id", traceID)
	if proof != "" { req.Header.Set("openai-sentinel-proof-token", proof) }
	if conduitToken != "" { req.Header.Set("X-Conduit-Token", conduitToken) }

	resp, err := c.Client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("conversation failed: %d, body: %s", resp.StatusCode, string(buf))
	}

	// 解析 SSE 获取 Conversation ID
	convID := ""
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "conversation_id") {
			var m map[string]interface{}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &m) == nil {
				if id, ok := m["conversation_id"].(string); ok { convID = id }
			}
		}
	}
	if convID == "" { return nil, fmt.Errorf("no conversation_id") }

	// 轮询并抓取图片
	return c.PollAndCaptureImages(convID)
}

func (c *OpenAIClient) PollAndCaptureImages(convID string) ([]string, error) {
	var finalUrls []string
	seen := make(map[string]bool)
	
	c.log("   [POLL] Capturing IMG2 assets from %s...\n", convID)

	for i := 0; i < 40; i++ {
		time.Sleep(5 * time.Second)
		req, _ := http.NewRequest("GET", "https://chatgpt.com/backend-api/conversation/"+convID, nil)
		c.setCommonHeaders(req)
		resp, err := c.Client.Do(req)
		if err != nil { continue }
		if resp.StatusCode == 429 { time.Sleep(10 * time.Second); continue }
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		foundNew := false
		matches := reAsset.FindAllString(string(body), -1)
		for _, asset := range matches {
			if !seen[asset] {
				seen[asset] = true
				u, err := c.GetDownloadURL(convID, asset)
				if err == nil && u != "" {
					finalUrls = append(finalUrls, u)
					foundNew = true
				}
			}
		}

		if len(finalUrls) > 0 && !foundNew {
			c.log("   ✅ Capture Complete. Total Assets: %d\n", len(finalUrls))
			return finalUrls, nil
		}
	}
	return nil, fmt.Errorf("poll timeout")
}

func (c *OpenAIClient) GetDownloadURL(convID, assetPointer string) (string, error) {
	fileID := strings.TrimPrefix(assetPointer, "file-service://")
	fileID = strings.TrimPrefix(fileID, "sediment://")
	apiURL := "https://chatgpt.com/backend-api/files/" + fileID + "/download"
	if strings.HasPrefix(assetPointer, "sediment://") {
		apiURL = fmt.Sprintf("https://chatgpt.com/backend-api/conversation/%s/attachment/%s/download", convID, fileID)
	}
	req, _ := http.NewRequest("GET", apiURL, nil)
	c.setCommonHeaders(req)
	resp, err := c.Client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	var out struct { DownloadURL string `json:"download_url"` }
	json.NewDecoder(resp.Body).Decode(&out)
	return out.DownloadURL, nil
}

func (c *OpenAIClient) DownloadImage(signedURL, outputPath string) error {
	req, _ := http.NewRequest("GET", signedURL, nil)
	if !strings.Contains(signedURL, "oaiusercontent.com") { c.setCommonHeaders(req) }
	resp, err := c.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return fmt.Errorf("failed %d", resp.StatusCode) }
	out, _ := os.Create(outputPath)
	defer out.Close()
	io.Copy(out, resp.Body)
	return nil
}

func (c *OpenAIClient) CheckCapability() (bool, error) {
	req, _ := http.NewRequest("GET", "https://chatgpt.com/backend-api/models", nil)
	c.setCommonHeaders(req)
	resp, err := c.Client.Do(req)
	if err != nil { return false, err }
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result struct { Models []struct { Slug string `json:"slug"` } `json:"models"` }
	json.NewDecoder(resp.Body).Decode(&result)
	for _, m := range result.Models {
		if m.Slug == "gpt-image-2" || m.Slug == "picture_v2" { return true, nil }
	}
	return false, nil
}
