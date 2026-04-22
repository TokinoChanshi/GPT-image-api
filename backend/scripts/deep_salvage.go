package main

import (
	"fmt"
	"os"
	"regexp"
)

func main() {
	data, _ := os.ReadFile("evo_image_api.db")
	
	// Sliding window extraction for very long tokens
	reAT := regexp.MustCompile(`eyJhbGciOiJSUzI1NiIs[A-Za-z0-9_-]{500,}`)
	reRT := regexp.MustCompile(`rt_[A-Za-z0-9_-]{30,}`)
	
	foundAT := make(map[string]bool)
	foundRT := make(map[string]bool)

	ats := reAT.FindAllString(string(data), -1)
	for _, m := range ats { foundAT[m] = true }

	rts := reRT.FindAllString(string(data), -1)
	for _, m := range rts { foundRT[m] = true }

	fmt.Printf("Recovered %d unique ATs and %d unique RTs.\n", len(foundAT), len(foundRT))
	
	f, _ := os.Create("../ultimate_recovery.txt")
	defer f.Close()
	
	for m := range foundAT { fmt.Fprintln(f, "AT:"+m) }
	for m := range foundRT { fmt.Fprintln(f, "RT:"+m) }
}
