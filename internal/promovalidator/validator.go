package promovalidator

import (
	"bufio"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// Config holds the validator rules and file locations.
type Config struct {
	Dir          string   // e.g. "./data"
	Files        []string // e.g. ["couponbase1.gz","couponbase2.gz","couponbase3.gz"]
	MinLen       int      // 8
	MaxLen       int      // 10
	RequiredHits int      // 2

	// Optional: cap the in-memory results cache (string->bool) to avoid unbounded growth.
	// If 0, a sensible default (50k) is used. If negative, caching is disabled.
	CacheSize int // e.g. 50000
}

// ValidatorService is the public interface for promo validation.
type ValidatorService interface {
	LoadCouponFiles() error
	ValidatePromoCode(code string) bool
}

// parallelValidator scans files on demand, concurrently, with early stop and bounded cache.
type parallelValidator struct {
	cfg  Config
	once sync.Once
	init error

	// bounded cache (LRU). If nil, caching disabled.
	lru *LRU
}

// NewValidatorService creates the validator with a bounded cache.
func NewValidatorService(cfg Config) ValidatorService {
	// default cache size if not provided
	size := cfg.CacheSize
	if size == 0 {
		size = 50000
	}
	var lru *LRU
	if size > 0 {
		lru = NewLRU(size)
	}
	return &parallelValidator{cfg: cfg, lru: lru}
}

// LoadCouponFiles validates configuration only (does NOT open files).
func (v *parallelValidator) LoadCouponFiles() error {
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

// ValidatePromoCode checks the code (case-sensitive) by scanning files in parallel.
// Missing/unreadable files are tolerated (skipped). Results are cached in a bounded LRU (if enabled).
func (v *parallelValidator) ValidatePromoCode(code string) bool {
	// Don’t block on config errors; best-effort.
	_ = v.LoadCouponFiles()

	code = strings.TrimSpace(code)
	if l := len(code); l < v.cfg.MinLen || l > v.cfg.MaxLen || !isAlnum(code) {
		return false
	}

	// Cache fast path
	if v.lru != nil {
		if val, ok := v.lru.Get(code); ok {
			return val
		}
	}

	// Parallel scan with early stop
	var found int32
	var wg sync.WaitGroup

	limit := runtime.NumCPU()
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)
	stop := atomic.Bool{} // set to true when we’ve reached RequiredHits

	for _, name := range v.cfg.Files {
		if stop.Load() {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		full := filepath.Join(v.cfg.Dir, name)

		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()

			if stop.Load() {
				return
			}
			ok, _ := foundInFile(path, code)
			if ok {
				if atomic.AddInt32(&found, 1) >= int32(v.cfg.RequiredHits) {
					stop.Store(true) // signal others to stop ASAP
				}
			}
		}(full)
	}
	wg.Wait()

	result := atomic.LoadInt32(&found) >= int32(v.cfg.RequiredHits)

	// Store in bounded cache
	if v.lru != nil {
		v.lru.Add(code, result)
	}
	return result
}

// foundInFile opens and streams the file, returning true iff the exact token (case-sensitive) is present.
// Missing/unreadable files are treated as "not found, nil error".
func foundInFile(path, code string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, nil // tolerate missing/unreadable file
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return false, nil // unreadable gzip; skip
		}
		defer gzr.Close()
		r = gzr
	}

	sc := bufio.NewScanner(r)
	// Allow long lines
	const maxLine = 1024 * 1024 // 1MB
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	for sc.Scan() {
		// tokenize by non-alphanumeric
		words := strings.FieldsFunc(sc.Text(), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		for _, w := range words {
			// exact, case-sensitive match; quick length check first
			if len(w) == len(code) && w == code {
				return true, nil
			}
		}
	}
	// scanner errors treated as "not found" but non-fatal
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
