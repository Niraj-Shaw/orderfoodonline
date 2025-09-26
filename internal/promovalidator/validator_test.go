package promovalidator

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// helper to create .gz file with lines
func writeGzipFile(t *testing.T, path string, lines []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	for _, l := range lines {
		if _, err := gz.Write([]byte(l + "\n")); err != nil {
			t.Fatalf("write gz: %v", err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
}

func TestValidatePromoCode_ExactMatch(t *testing.T) {
	dir := t.TempDir()
	writeGzipFile(t, filepath.Join(dir, "couponbase1.gz"), []string{"HAPPYHRS", "WELCOME10"})

	cfg := Config{
		Dir:          dir,
		Files:        []string{"couponbase1.gz"},
		MinLen:       8,
		MaxLen:       10,
		RequiredHits: 1,
	}
	v := NewValidatorService(cfg)

	tests := []struct {
		code string
		want bool
	}{
		{"HAPPYHRS", true},            // exact match
		{"WELCOME10", true},           // exact match
		{"happyhrs", false},           // case sensitive
		{"WELCOME", false},            // too short
		{"WELCOME1000", false},        // too long
		{"NOTEXIST", false},           // not present
		{"randomHAPPYHRStext", false}, // substring only
	}
	for _, tt := range tests {
		if got := v.ValidatePromoCode(tt.code); got != tt.want {
			t.Errorf("ValidatePromoCode(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestValidatePromoCode_SubstringVsToken(t *testing.T) {
	dir := t.TempDir()
	// file contains line with "random HAPPYHRS here"
	writeGzipFile(t, filepath.Join(dir, "couponbase1.gz"), []string{"random HAPPYHRS here"})

	cfg := Config{
		Dir:          dir,
		Files:        []string{"couponbase1.gz"},
		MinLen:       8,
		MaxLen:       10,
		RequiredHits: 1,
	}
	v := NewValidatorService(cfg)

	if !v.ValidatePromoCode("HAPPYHRS") {
		t.Fatalf("expected HAPPYHRS to be valid when tokenized in middle of line")
	}
	if v.ValidatePromoCode("randomHAPPYHRS") {
		t.Fatalf("expected concatenated substring not to match")
	}
}

func TestValidatePromoCode_ConfigErrors(t *testing.T) {
	dir := t.TempDir()
	writeGzipFile(t, filepath.Join(dir, "file.gz"), []string{"HAPPYHRS"})

	bad := []Config{
		{Dir: dir, Files: nil, MinLen: 8, MaxLen: 10, RequiredHits: 1},                 // no files
		{Dir: dir, Files: []string{"file.gz"}, MinLen: 0, MaxLen: 10, RequiredHits: 1}, // invalid min
		{Dir: dir, Files: []string{"file.gz"}, MinLen: 8, MaxLen: 5, RequiredHits: 1},  // invalid range
		{Dir: dir, Files: []string{"file.gz"}, MinLen: 8, MaxLen: 10, RequiredHits: 0}, // invalid hits
	}
	for _, cfg := range bad {
		v := NewValidatorService(cfg)
		if err := v.LoadCouponFiles(); err == nil {
			t.Errorf("expected error for cfg: %+v", cfg)
		}
	}
}

func TestValidatePromoCode_MissingFilesAreIgnored(t *testing.T) {
	dir := t.TempDir()
	writeGzipFile(t, filepath.Join(dir, "couponbase1.gz"), []string{"HAPPYHRS"})
	writeGzipFile(t, filepath.Join(dir, "couponbase2.gz"), []string{"random HAPPYHRS here"})
	// couponbase3.gz intentionally missing

	cfg := Config{
		Dir:          dir,
		Files:        []string{"couponbase1.gz", "couponbase2.gz", "couponbase3.gz"},
		MinLen:       8,
		MaxLen:       10,
		RequiredHits: 2,
	}
	v := NewValidatorService(cfg)

	if !v.ValidatePromoCode("HAPPYHRS") {
		t.Fatalf("expected HAPPYHRS to be valid across the two existing files")
	}
}

func TestValidatePromoCode_CaseSensitive(t *testing.T) {
	dir := t.TempDir()
	writeGzipFile(t, filepath.Join(dir, "couponbase1.gz"), []string{"HAPPYHRS"})
	writeGzipFile(t, filepath.Join(dir, "couponbase2.gz"), []string{"HAPPYHRS"})

	cfg := Config{
		Dir:          dir,
		Files:        []string{"couponbase1.gz", "couponbase2.gz"},
		MinLen:       8,
		MaxLen:       10,
		RequiredHits: 2,
	}
	v := NewValidatorService(cfg)

	if v.ValidatePromoCode("happyhrs") {
		t.Fatalf("expected lowercase happyhrs to be invalid (case-sensitive check)")
	}
}
