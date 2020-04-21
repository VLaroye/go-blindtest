package main

import (
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"log"
	"regexp"
	"strings"
	"unicode"
)

func sanitizeString(s string) string {
	sanitizedString := s
	// Lowercase
	sanitizedString = strings.ToLower(sanitizedString)
	// Remove spaces
	sanitizedString = strings.ReplaceAll(sanitizedString, " ", "")
	// Remove accents
	sanitizedString = removeAccents(sanitizedString)
	// Remove special chars
	sanitizedString = removeNonAlphanumericChars(sanitizedString)

	return sanitizedString
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, e := transform.String(t, s)
	if e != nil {
		panic(e)
	}
	return output
}

func removeNonAlphanumericChars(s string) string {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(s, "")
}
