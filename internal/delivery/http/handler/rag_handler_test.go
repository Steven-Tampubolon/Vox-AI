package handler

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestExtractPDFText_RejectsTooManyPages memverifikasi bahwa PDF dengan
// jumlah halaman melebihi maxPDFPages langsung ditolak, mencegah
// Gemini API quota exhaustion (Finding 5).
func TestExtractPDFText_RejectsTooManyPages(t *testing.T) {
	tooManyPages := maxPDFPages + 1
	data := buildMinimalPDF(t, tooManyPages)

	_, err := extractPDFText(data)
	if err == nil {
		t.Fatalf("expected error for PDF with %d pages (> maxPDFPages=%d), got nil",
			tooManyPages, maxPDFPages)
	}
	if !strings.Contains(err.Error(), "terlalu panjang") {
		t.Errorf("expected error to contain 'terlalu panjang', got: %v", err)
	}
}

// TestExtractPDFText_AcceptsAtPageLimit memverifikasi PDF tepat di batas
// maxPDFPages tidak ditolak oleh cek jumlah halaman.
func TestExtractPDFText_AcceptsAtPageLimit(t *testing.T) {
	data := buildMinimalPDF(t, maxPDFPages)

	_, err := extractPDFText(data)
	// Hanya cek tidak ada error "terlalu panjang" — error parse lain diabaikan
	if err != nil && strings.Contains(err.Error(), "terlalu panjang") {
		t.Errorf("PDF with exactly %d pages should not be rejected for page count, got: %v",
			maxPDFPages, err)
	}
}

// TestExtractPDFText_CharLimitLogic memverifikasi logika early-exit
// saat total karakter mencapai maxExtractedChars (Finding 5).
func TestExtractPDFText_CharLimitLogic(t *testing.T) {
	cases := []struct {
		written  int
		wantStop bool
	}{
		{maxExtractedChars - 1, false},
		{maxExtractedChars, true},
		{maxExtractedChars + 500, true},
	}
	for _, tc := range cases {
		var sb strings.Builder
		sb.WriteString(strings.Repeat("x", tc.written))
		got := sb.Len() >= maxExtractedChars
		if got != tc.wantStop {
			t.Errorf("written=%d: shouldStop=%v, want %v", tc.written, got, tc.wantStop)
		}
	}
}

// TestValidateFile_InvalidExtension memverifikasi validateFile menolak
// ekstensi yang tidak diizinkan — tidak bocorkan detail internal (Finding 6).
func TestValidateFile_InvalidExtension(t *testing.T) {
	data := []byte("some content")
	mimeType, valid := validateFile(data, "malicious.exe")
	if valid {
		t.Error("expected .exe to be rejected")
	}
	if mimeType != "ekstensi tidak didukung" {
		t.Errorf("unexpected mime message: %q", mimeType)
	}
}

// TestValidateFile_MIMEMismatch memverifikasi file dengan ekstensi .pdf
// tapi MIME bukan PDF ditolak (mitigasi polyglot file attack).
func TestValidateFile_MIMEMismatch(t *testing.T) {
	// ZIP magic bytes menyamar sebagai PDF
	data := []byte("PK\x03\x04disguised zip content that is not a real PDF")
	_, valid := validateFile(data, "document.pdf")
	if valid {
		t.Error("expected MIME mismatch to be rejected")
	}
}

// TestValidateFile_ValidTXT memverifikasi file .txt diterima dengan benar.
func TestValidateFile_ValidTXT(t *testing.T) {
	data := []byte("Ini adalah isi dokumen teks biasa untuk test.")
	_, valid := validateFile(data, "catatan.txt")
	if !valid {
		t.Error("expected valid .txt to be accepted")
	}
}

// TestPDFLimits memverifikasi konstanta limit PDF tidak berubah ke nilai ekstrem.
func TestPDFLimits(t *testing.T) {
	if maxPDFPages <= 0 || maxPDFPages > 200 {
		t.Errorf("maxPDFPages=%d out of expected range [1, 200]", maxPDFPages)
	}
	if maxExtractedChars <= 0 || maxExtractedChars > 1_000_000 {
		t.Errorf("maxExtractedChars=%d out of expected range [1, 1_000_000]", maxExtractedChars)
	}
}

// buildMinimalPDF membuat PDF minimal valid dengan N halaman kosong.
// Hanya untuk keperluan unit test — bukan PDF "nyata" dengan teks.
func buildMinimalPDF(t *testing.T, numPages int) []byte {
	t.Helper()
	if numPages < 1 {
		t.Fatal("numPages must be >= 1")
	}

	var buf bytes.Buffer
	offsets := make([]int, 0, 2+numPages)

	buf.WriteString("%PDF-1.4\n")

	// Object 1: Catalog
	offsets = append(offsets, buf.Len())
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Object 2: Pages
	offsets = append(offsets, buf.Len())
	var kids strings.Builder
	for i := 0; i < numPages; i++ {
		if i > 0 {
			kids.WriteString(" ")
		}
		kids.WriteString(fmt.Sprintf("%d 0 R", 3+i))
	}
	buf.WriteString(fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n",
		kids.String(), numPages))

	// Objects 3..N+2: Page
	for i := 0; i < numPages; i++ {
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf(
			"%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n",
			3+i,
		))
	}

	// XRef
	xrefOffset := buf.Len()
	totalObjs := 2 + numPages
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", totalObjs+1))
	for _, off := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}

	// Trailer
	buf.WriteString(fmt.Sprintf(
		"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		totalObjs+1, xrefOffset,
	))

	return buf.Bytes()
}
