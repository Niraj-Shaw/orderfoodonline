package promovalidator

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create a temporary test file.
func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
	defer f.Close()
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to test file %s: %v", path, err)
	}
	return path
}

// Helper function to create a temporary gzipped test file.
func createGzipTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create gzip test file %s: %v", path, err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte(content))
	if err != nil {
		t.Fatalf("Failed to write to gzip test file %s: %v", path, err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer for %s: %v", path, err)
	}
	return path
}

func TestValidatePromoCode(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy coupon files
	createTestFile(t, tempDir, "coupons1.txt", "VALID12345, ANOTHERONE, IGNORETHIS")
	createGzipTestFile(t, tempDir, "coupons2.gz", "VALID12345 ALSOVALID")
	createTestFile(t, tempDir, "coupons3.txt", "THIRDMATCH ALSOVALID")

	cfg := Config{
		Dir:          tempDir,
		Files:        []string{"coupons1.txt", "coupons2.gz", "coupons3.txt"},
		MinLen:       8,
		MaxLen:       10,
		RequiredHits: 2,
	}

	validator := NewValidatorService(cfg)

	testCases := []struct {
		name string
		code string
		want bool
	}{
		{"Valid code, in 2 files", "VALID12345", true},
		{"Valid code, in 2 files", "ALSOVALID", true},
		{"Valid code, but only in 1 file", "ANOTHERONE", false},
		{"Valid code, but only in 1 file", "THIRDMATCH", false},
		{"Code too short", "SHORT", false},
		{"Code too long", "WAYTOOLONGCODE", false},
		{"Non-alphanumeric code", "INVALID-!@", false},
		{"Code not in any file", "NONEXISTENT", false},
		{"Empty code", "", false},
		{"Code with leading/trailing space", "  VALID12345  ", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := validator.ValidatePromoCode(tc.code)
			if got != tc.want {
				t.Errorf("ValidatePromoCode(%q) = %v; want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestLoadCouponFiles_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "No files configured",
			cfg:     Config{Dir: tempDir, Files: []string{}},
			wantErr: "validator: no files configured",
		},
		{
			name:    "Invalid length bounds",
			cfg:     Config{Dir: tempDir, Files: []string{"a.txt"}, MinLen: 10, MaxLen: 8},
			wantErr: "validator: invalid length bounds",
		},
		{
			name:    "Invalid RequiredHits",
			cfg:     Config{Dir: tempDir, Files: []string{"a.txt"}, MinLen: 1, MaxLen: 8, RequiredHits: 0},
			wantErr: "validator: RequiredHits must be >= 1",
		},
		{
			name: "File not found",
			cfg: Config{
				Dir:          tempDir,
				Files:        []string{"nonexistent.txt"},
				MinLen:       8,
				MaxLen:       10,
				RequiredHits: 1,
			},
			wantErr: "open " + filepath.Join(tempDir, "nonexistent.txt") + ": no such file or directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := NewValidatorService(tc.cfg)
			err := validator.LoadCouponFiles()
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("LoadCouponFiles() error = %v; wantErr %q", err, tc.wantErr)
			}
			// Also test that ValidatePromoCode returns false when loading fails
			if validator.ValidatePromoCode("ANYCODE") {
				t.Error("ValidatePromoCode() should return false when loading fails, but got true")
			}
		})
	}
}

func TestIsAlnum(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want bool
	}{
		{"Lowercase", "abcdef", true},
		{"Uppercase", "ABCDEF", true},
		{"Numbers", "123456", true},
		{"Mixed", "Abc123Def456", true},
		{"With space", "abc def", false},
		{"With hyphen", "abc-def", false},
		{"With punctuation", "abc.def!", false},
		{"Empty string", "", true}, // isAlnum returns true for empty string as the loop doesn't run
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAlnum(tc.in); got != tc.want {
				t.Errorf("isAlnum(%q) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoadCouponsFromFile(t *testing.T) {
	tempDir := t.TempDir()
	path := createTestFile(t, tempDir, "test.txt", "  word1, (word2) word3!  \nword4;word5. ")

	set, err := loadCouponsFromFile(path, 5, 5)
	if err != nil {
		t.Fatalf("loadCouponsFromFile() returned an unexpected error: %v", err)
	}

	expected := map[string]bool{
		"word1": true,
		"word2": true,
		"word3": true,
		"word4": true,
		"word5": true,
	}

	if len(set) != len(expected) {
		t.Errorf("Expected %d items, but got %d. Set: %v", len(expected), len(set), set)
	}

	for k := range expected {
		if !set[k] {
			t.Errorf("Expected key %q to be in the set", k)
		}
	}

	// Test non-existent file
	_, err = loadCouponsFromFile("nonexistent.txt", 1, 10)
	if err == nil {
		t.Error("Expected an error for non-existent file, but got nil")
	}
}
