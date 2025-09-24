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

// validatorService implements ValidatorService.
type validatorService struct {
	cfg        Config
	couponSets []map[string]bool
	once       sync.Once
	initErr    error
}

// NewValidatorService creates a validator instance with the given config.
func NewValidatorService(cfg Config) ValidatorService {
	return &validatorService{
		cfg:        cfg,
		couponSets: make([]map[string]bool, len(cfg.Files)),
	}
}

func (s *validatorService) LoadCouponFiles() error {
	s.once.Do(func() {
		if len(s.cfg.Files) == 0 {
			s.initErr = errors.New("validator: no files configured")
			return
		}
		if s.cfg.MinLen <= 0 || s.cfg.MaxLen < s.cfg.MinLen {
			s.initErr = errors.New("validator: invalid length bounds")
			return
		}
		if s.cfg.RequiredHits <= 0 {
			s.initErr = errors.New("validator: RequiredHits must be >= 1")
			return
		}
		for i, name := range s.cfg.Files {
			full := filepath.Join(s.cfg.Dir, name)
			set, err := loadCouponsFromFile(full, s.cfg.MinLen, s.cfg.MaxLen)
			if err != nil {
				s.initErr = err
				return
			}
			s.couponSets[i] = set
		}
	})
	return s.initErr
}

func (s *validatorService) ValidatePromoCode(code string) bool {
	if err := s.LoadCouponFiles(); err != nil { // safe; Once prevents re-run
		return false
	}
	trimmed := strings.TrimSpace(code)
	if l := len(trimmed); l < s.cfg.MinLen || l > s.cfg.MaxLen || !isAlnum(trimmed) {
		return false
	}
	found := 0
	for _, set := range s.couponSets {
		if set[trimmed] {
			found++
			if found >= s.cfg.RequiredHits {
				return true
			}
		}
	}
	return false
}

// --- helpers ---

func loadCouponsFromFile(filename string, minLen, maxLen int) (map[string]bool, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(filename), ".gz") {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gzr.Close()
		r = gzr
	}

	set := make(map[string]bool, 1024)
	sc := bufio.NewScanner(r)

	// Increase scanner buffer size to handle long lines.
	const maxLine = 1024 * 1024 // 1 MB per line
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	for sc.Scan() {
		words := strings.FieldsFunc(sc.Text(), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		for _, w := range words {
			if l := len(w); l >= minLen && l <= maxLen && isAlnum(w) {
				set[w] = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return set, nil
}

func isAlnum(s string) bool {
	for _, ch := range s {
		if !(ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}
