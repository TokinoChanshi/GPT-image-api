package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/sha3"
)

const (
	powPrefixRequirements = "gAAAAAC"
	powPrefixProof        = "gAAAAAB"
	requirementsDifficulty = "0fffff"
	maxRequirementsIter   = 500000
	maxProofIter          = 100000
)

var (
	powCores   = []int{16, 24, 32}
	powScreens = []int{3000, 4000, 6000}
	powNavKeys = []string{
		"webdriver:false", "vendor:Google Inc.", "cookieEnabled:true",
		"pdfViewerEnabled:true", "hardwareConcurrency:12",
		"language:en-US", "mimeTypes:[object MimeTypeArray]",
		"userAgentData:[object NavigatorUAData]",
	}
	powWinKeys = []string{
		"innerWidth", "innerHeight", "devicePixelRatio", "screen",
		"chrome", "location", "history", "navigator",
	}
	powReactListeners = []string{"_reactListeningcfilawjnerp", "_reactListening9ne2dfo1i47"}
	powProofEvents    = []string{"alert", "ontransitionend", "onprogress"}
	perfCounter       uint64
)

type POWConfig struct {
	userAgent string
	arr       [18]interface{}
}

func NewPOWConfig(userAgent string) *POWConfig {
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}
	now := time.Now().UTC()
	timeStr := now.Format("Mon Jan 02 2006 15:04:05") + " GMT+0000 (UTC)"
	
	// Simple random source
	r := time.Now().UnixNano()
	perf := float64(atomic.AddUint64(&perfCounter, 1)) + float64(r%1000)/1000.0

	c := &POWConfig{userAgent: userAgent}
	c.arr = [18]interface{}{
		powCores[int(r%int64(len(powCores)))] + powScreens[int(r%int64(len(powScreens)))],
		timeStr,
		nil,
		float64(r%1000000)/1000000.0,
		userAgent,
		nil,
		"dpl=1440a687921de39ff5ee56b92807faaadce73f13",
		"en-US",
		"en-US,zh-CN",
		0,
		powNavKeys[int(r%int64(len(powNavKeys)))],
		"location",
		powWinKeys[int(r%int64(len(powWinKeys)))],
		perf,
		GenerateUUID(),
		"",
		8,
		now.Unix(),
	}
	return c
}

func (c *POWConfig) RequirementsToken() string {
	seed := strconv.FormatFloat(float64(time.Now().UnixNano()%1000000)/1000000.0, 'f', -1, 64)
	b64, ok := c.solveRequirements(seed, requirementsDifficulty)
	if !ok {
		return powPrefixRequirements + "FAILED"
	}
	return powPrefixRequirements + b64
}

func (c *POWConfig) solveRequirements(seed, difficulty string) (string, bool) {
	target, _ := hex.DecodeString(difficulty)
	diffLen := len(difficulty)

	arr := c.arr
	head, _ := json.Marshal([]interface{}{arr[0], arr[1], arr[2]})
	p1 := append(head[:len(head)-1], ',')

	mid, _ := json.Marshal([]interface{}{arr[4], arr[5], arr[6], arr[7], arr[8]})
	p2 := make([]byte, 0, len(mid)+2)
	p2 = append(p2, ',')
	p2 = append(p2, mid[1:len(mid)-1]...)
	p2 = append(p2, ',')

	tail, _ := json.Marshal([]interface{}{
		arr[10], arr[11], arr[12], arr[13], arr[14], arr[15], arr[16], arr[17],
	})
	p3 := make([]byte, 0, len(tail)+1)
	p3 = append(p3, ',')
	p3 = append(p3, tail[1:]...)

	hasher := sha3.New512()
	seedB := []byte(seed)

	for i := 0; i < maxRequirementsIter; i++ {
		d1 := strconv.Itoa(i)
		d2 := strconv.Itoa(i >> 1)

		var buf bytes.Buffer
		buf.Write(p1)
		buf.WriteString(d1)
		buf.Write(p2)
		buf.WriteString(d2)
		buf.Write(p3)

		b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
		hasher.Reset()
		hasher.Write(seedB)
		hasher.Write([]byte(b64))
		sum := hasher.Sum(nil)

		cmpLen := diffLen
		if cmpLen > len(sum) { cmpLen = len(sum) }
		if cmpLen > len(target) { cmpLen = len(target) }

		if bytes.Compare(sum[:cmpLen], target[:cmpLen]) <= 0 {
			return b64, true
		}
	}
	return "", false
}

func SolveProofToken(seed, difficulty, userAgent string) string {
	if seed == "" || difficulty == "" { return "" }
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	}
	
	r := time.Now().UnixNano()
	screen := powScreens[int(r%int64(len(powScreens)))] * (1 << (r % 3))
	timeStr := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	proofConfig := []interface{}{
		screen,
		timeStr,
		nil,
		0,
		userAgent,
		"https://tcr9i.chat.openai.com/v2/35536E1E-65B4-4D96-9D97-6ADB7EFF8147/api.js",
		"dpl=1440a687921de39ff5ee56b92807faaadce73f13",
		"en",
		"en-US",
		nil,
		"plugins:[object PluginArray]",
		powReactListeners[int(r%int64(len(powReactListeners)))],
		powProofEvents[int(r%int64(len(powProofEvents)))],
	}

	diffLen := len(difficulty)
	hasher := sha3.New512()
	for i := 0; i < maxProofIter; i++ {
		proofConfig[3] = i
		raw, _ := json.Marshal(proofConfig)
		b64 := base64.StdEncoding.EncodeToString(raw)
		hasher.Reset()
		hasher.Write([]byte(seed + b64))
		sum := hasher.Sum(nil)
		hexStr := hex.EncodeToString(sum)
		if strings.Compare(hexStr[:diffLen], difficulty) <= 0 {
			return powPrefixProof + b64
		}
	}
	return ""
}
