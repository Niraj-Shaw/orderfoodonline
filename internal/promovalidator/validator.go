// internal/promovalidator/validator.go
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
	Dir                      string   // e.g. "./data"
	Files                    []string // e.g. ["couponbase1.gz", "couponbase2.gz", "couponbase3.gz"]
	MinLen                   int      // minimum code length (e.g. 8)
	MaxLen                   int      // maximum code length (e.g. 10)
	RequiredHits             int      // how many different files the code must appear in (e.g. 2)
	MaxConcurrentValidations int      // If <= 0, a small default (2) is used.
}

// ValidatorService is the public interface for promo validation.
type ValidatorService interface {
	// LoadCouponFiles validates the configuration once. (Does NOT open/parse files.)
	LoadCouponFiles() error
	// ValidatePromoCode checks a code against the configured files.
	// Case-sensitive, streams files on demand, tolerates missing/unreadable files,
	// and returns true as soon as RequiredHits is reached.
	ValidatePromoCode(code string) bool
}

// streamingValidator implements ValidatorService with on-demand streaming and caching.
type streamingValidator struct {
	cfg  Config
	once sync.Once
	init error

	// tiny concurrent cache: promoCode -> bool (result).
	cache sync.Map

	// semaphore to cap concurrent validations (protects memory/disk IO under load)
	sem chan struct{}
}

// NewValidatorService creates a streaming validator with an optional concurrency cap.
func NewValidatorService(cfg Config) ValidatorService {
	max := cfg.MaxConcurrentValidations
	if max <= 0 {
		max = 2 // small, safe default
	}
	return &streamingValidator{
		cfg: cfg,
		sem: make(chan struct{}, max),
	}
}

// LoadCouponFiles validates configuration only (no file IO here).
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
	})
	return v.init
}

// ValidatePromoCode checks code (case-sensitive), scanning files on demand.
// Missing/unreadable files are skipped. Results are cached per code.
func (v *streamingValidator) ValidatePromoCode(code string) bool {
	// Concurrency cap for validations
	v.sem <- struct{}{}
	defer func() { <-v.sem }()

	// Best-effort config validation (Once); ignore error at call site.
	_ = v.LoadCouponFiles()

	code = strings.TrimSpace(code)
	if l := len(code); l < v.cfg.MinLen || l > v.cfg.MaxLen || !isAlnum(code) {
		return false
	}

	// Cache fast path
	if cached, ok := v.cache.Load(code); ok {
		return cached.(bool)
	}

	// Stream files sequentially with early exit on RequiredHits
	found := 0
	for _, name := range v.cfg.Files {
		full := filepath.Join(v.cfg.Dir, name)
		ok, err := foundInFile(full, code, v.cfg.MinLen, v.cfg.MaxLen)
		if err != nil {
			// tolerate unreadable/missing file: skip
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
// If file can't be opened or scan fails, it returns (false, nil) to indicate "not found, but not fatal".
func foundInFile(filename, code string, minLen, maxLen int) (bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		// treat missing/unreadable file as non-fatal
		return false, nil
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(filename), ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			// unreadable gzip -> skip
			return false, nil
		}
		defer gzr.Close()
		r = gzr
	}

	sc := bufio.NewScanner(r)
	// allow reasonably long lines (1MiB max token buffer)
	const maxLine = 1024 * 1024
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	for sc.Scan() {
		// split into tokens by non-alphanumeric
		words := strings.FieldsFunc(sc.Text(), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		for _, w := range words {
			if len(w) == len(code) && len(w) >= minLen && len(w) <= maxLen && isAlnum(w) && w == code {
				return true, nil
			}
		}
	}
	// treat scanner errors as "not found" but non-fatal
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
