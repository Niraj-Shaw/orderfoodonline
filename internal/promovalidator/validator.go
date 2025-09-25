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
}

// ValidatorService is the public interface for promo validation.
type ValidatorService interface {
	LoadCouponFiles() error
	ValidatePromoCode(code string) bool
}

// parallelValidator implements ValidatorService.
// It scans files on demand, but does so concurrently and short-circuits early.
type parallelValidator struct {
	cfg  Config
	once sync.Once
	init error

	// tiny result cache: code -> bool
	cache sync.Map
}

// NewValidatorService creates the validator.
func NewValidatorService(cfg Config) ValidatorService {
	return &parallelValidator{cfg: cfg}
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

// ValidatePromoCode checks code (case-sensitive) by scanning files in parallel.
// Missing/unreadable files are tolerated (skipped). Results are cached.
func (v *parallelValidator) ValidatePromoCode(code string) bool {
	// Best-effort config validation (don’t block lookups).
	_ = v.LoadCouponFiles()

	code = strings.TrimSpace(code)
	if l := len(code); l < v.cfg.MinLen || l > v.cfg.MaxLen || !isAlnum(code) {
		return false
	}

	// cache fast path
	if cached, ok := v.cache.Load(code); ok {
		return cached.(bool)
	}

	// parallel scan with early stop
	var found int32
	var wg sync.WaitGroup

	// Goroutine limiter: avoid spawning too many (min(NumCPU, len(files))).
	limit := runtime.NumCPU()
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)

	stop := atomic.Bool{} // when true, workers should return ASAP

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

			// If someone already met the threshold, bail early.
			if stop.Load() {
				return
			}

			ok, _ := foundInFile(path, code, v.cfg.MinLen, v.cfg.MaxLen)
			if ok {
				if atomic.AddInt32(&found, 1) >= int32(v.cfg.RequiredHits) {
					// signal others to stop as soon as they can
					stop.Store(true)
				}
			}
		}(full)
	}

	wg.Wait()

	result := atomic.LoadInt32(&found) >= int32(v.cfg.RequiredHits)
	v.cache.Store(code, result)
	return result
}

// foundInFile opens and streams the file, returning true iff the exact token (case-sensitive) is present.
// Missing/unreadable files are treated as "not found, nil error".
func foundInFile(path, code string, minLen, maxLen int) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, nil // tolerate missing or unreadable file
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
			// exact, case-sensitive match; length already bounded by caller’s code,
			// but keep quick length check to skip needless compares:
			if len(w) == len(code) && w == code {
				return true, nil
			}
		}
	}
	// scanner errors are treated as "not found" but non-fatal
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
