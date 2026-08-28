package utils

import (
	"github.com/k3a/html2text"
)

// Cleaned Passed in email body and return clean text, only, strip out, html tags e.t.c
func CleanHTMLToText(emailBody string) string {
	plainText := html2text.HTML2Text(emailBody)
	if len(plainText) <= 1 {
		return ""
	} else {
		return plainText
	}
}
