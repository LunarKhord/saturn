package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func ConvertMp3ToOGG(filePathMP3 string) (string, error) {
	fmt.Println("[AudioConverter]: Started....")

	oggFilename := fmt.Sprintf("reply_%d.ogg", time.Now().Unix())
	oggPath := filepath.Join("public", "replies", oggFilename)

	fmt.Println("Converting MP3 to OGG/Opus via FFmpeg...")

	// 🚨 FIX 1: Use filePathMP3 instead of "filepath" to avoid clashing with the package name
	cmd := exec.Command("ffmpeg", "-y", "-i", filePathMP3, "-c:a", "libopus", "-ac", "1", "-ar", "48000", oggPath)

	// Capture FFmpeg output in case it fails
	var out bytes.Buffer
	cmd.Stderr = &out

	// 🚨 FIX 2: Declare err with :=
	err := cmd.Run()
	if err != nil {
		fmt.Printf("FFmpeg conversion failed: %v | Output: %s\n", err, out.String())
		// 🚨 FIX 3: Actually return the error
		return "nil", fmt.Errorf("FFmpeg conversion failed: %v", err)
	}

	// Clean up the temporary MP3 file
	os.Remove(filePathMP3)
	fmt.Println("✅ Successfully converted to WhatsApp-compatible OGG/Opus!")
	fmt.Println("The ogg path is now:", oggPath)

	return oggPath, nil
}
