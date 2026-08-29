package bucket

import (
	"bytes"
	"fmt"
	"os"
	"time"

	storage_go "github.com/supabase-community/storage-go"
)

func UploadVoiceToSupabase(oggBytes []byte) (string, error) {
	supabaseURL := "https://joozxtnshnyrdwreyoud.supabase.co/storage/v1"
	serviceRoleKey := ""

	// Initialize the client
	storageClient := storage_go.NewClient(supabaseURL, serviceRoleKey, nil)

	// 1. Getting the bucket info (Using a uniquely named variable to avoid conflicts)
	bucketInfo, err := storageClient.GetBucket("StaturnUploadsSpeech")
	if err != nil {
		fmt.Println("An error occurred:", err)
		os.Exit(1)
	}

	fmt.Println("✅ Successfully connected to Supabase Storage!")
	fmt.Printf("Bucket Name: %s\n", bucketInfo.Name)
	fmt.Printf("Is Public: %v\n", bucketInfo.Public)

	// 2. Create a unique file path inside the bucket
	fileName := fmt.Sprintf("replies/reply_%d.ogg", time.Now().Unix())

	// 3. Convert []byte to io.Reader using bytes.NewReader!
	fileReader := bytes.NewReader(oggBytes)

	fmt.Println("🚀 Uploading to Supabase Storage...")

	// 4. Upload the file.
	// We use `uploadResult` (a new variable) and `:=` because `uploadResult` is new,
	// even though `err` is being reassigned. This is perfectly valid Go.
	uploadResult, err := storageClient.UploadFile("StaturnUploadsSpeech", fileName, fileReader)
	if err != nil {
		return "", fmt.Errorf("Supabase upload failed: %v", err)
	}

	fmt.Println("✅ Supabase upload successful. File key:", uploadResult.Key)

	// 5. 🚨 FIX THE MISSING RETURN: Construct and return the public URL
	// The standard format for a public Supabase Storage object URL
	publicURL := fmt.Sprintf("https://joozxtnshnyrdwreyoud.supabase.co/storage/v1/object/public/StaturnUploadsSpeech/%s", fileName)

	return publicURL, nil
}
