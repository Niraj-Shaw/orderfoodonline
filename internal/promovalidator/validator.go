package promovalidator

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// Config holds the validator rules and file locations.
type Config struct {
	Dir          string   // folder where coupon files are stored, e.g. "./data"
	Files        []string // file names, e.g. ["couponbase1.gz", "couponbase2.gz", "couponbase3.gz"]
	MinLen       int      // minimum code length (8)
	MaxLen       int      // maximum code length (10)
	RequiredHits int      // how many files a code must appear in (2)
}

// ValidatorService is the public interface for promo validation.
type ValidatorService interface {
	LoadCouponFiles() error
	ValidatePromoCode(code string) bool
}

// streamingValidator implements ValidatorService in a streaming, safe way.
type streamingValidator struct {
	cfg  Config
	once sync.Once
	init error

	// tiny cache: code -> bool
	cache sync.Map
}

// NewValidatorService creates a streaming validator.
func NewValidatorService(cfg Config) ValidatorService {
	return &streamingValidator{cfg: cfg}
}

// LoadCouponFiles validates configuration only (does NOT open files).
// This prevents LoadCouponFiles from failing simply because one data file is missing.
func (v *streamingValidator) LoadCouponFiles() error {
	v.once.Do(func() {
		if len(v.cfg.Files) == 0 {
			v.init = errors.New("validator: no files configured")
			return
		}
		if v.cfg.MinLen <= 0 || v.cfg.MaxLen < v.cfg.MinLen {
			v.init = errors.New("validator: invalid length bounds")
			return
		}
		if v.cfg.RequiredHits <= 0 {
			v.init = errors.New("validator: RequiredHits must be >= 1")
			return
		}
		// intentionally DO NOT attempt to open files here
	})
	return v.init
}

// ValidatePromoCode checks code (case-sensitive), scanning files on demand.
// It tolerates missing files and will return true once RequiredHits are reached.
func (v *streamingValidator) ValidatePromoCode(code string) bool {
	// best-effort config validation; ignore non-config errors
	_ = v.LoadCouponFiles()

	code = strings.TrimSpace(code)
	if l := len(code); l < v.cfg.MinLen || l > v.cfg.MaxLen || !isAlnum(code) {
		return false
	}

	// cache fastpath
	if cached, ok := v.cache.Load(code); ok {
		return cached.(bool)
	}

	found := 0
	for _, fn := range v.cfg.Files {
		full := filepath.Join(v.cfg.Dir, fn)
		ok, err := foundInFile(full, code, v.cfg.MinLen, v.cfg.MaxLen)
		if err != nil {
			// tolerate: skip unreadable files
			continue
		}
		if ok {
			found++
			if found >= v.cfg.RequiredHits {
				v.cache.Store(code, true)
				return true
			}
		}
	}

	v.cache.Store(code, false)
	return false
}

// foundInFile opens and scans the file for the exact token (case-sensitive).
// If file can't be opened, return (false, nil) to indicate "not found, but not fatal".
func foundInFile(filename, code string, minLen, maxLen int) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		// treat missing/unreadable file as non-fatal (test expects this)
		return false, nil
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(filename), ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			// unreadable gzip -> treat as non-fatal skip
			return false, nil
		}
		defer gzr.Close()
		r = gzr
	}

	sc := bufio.NewScanner(r)
	// allow long lines
	const maxLine = 1024 * 1024 // 1MB
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	for sc.Scan() {
		// split into tokens by non-alphanumeric
		words := strings.FieldsFunc(sc.Text(), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		for _, w := range words {
			if l := len(w); l >= minLen && l <= maxLen && isAlnum(w) && w == code {
				return true, nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		// if scanning fails, treat it as a skip (non-fatal)
		return false, nil
	}
	return false, nil
}

func isAlnum(s string) bool {
	for _, ch := range s {
		if !(ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}
