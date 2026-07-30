package importer

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// meisaiLineRe matches a transaction line from a Yucho Bank ("ゆうちょ銀行") credit card
// statement PDF (ご利用明細書 / "meisai"), e.g.:
//
//	26/05/04 ビックカメラ         有楽町店                     JPY     15,920.00
var meisaiLineRe = regexp.MustCompile(`^(\d{2}/\d{2}/\d{2})\s+(.+?)\s+JPY\s+(-?[\d,]+\.\d{2})\s*$`)

// ReadPDF extracts transaction rows from a meisai-style statement PDF, returning the
// same (headers, rows) shape as ReadCSV (rows includes the header row at index 0) so
// it can be parsed via the existing ParseRows/Mapping flow unchanged.
func ReadPDF(r io.Reader) (headers []string, rows [][]string, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, nil, fmt.Errorf("not a PDF file")
	}

	cmd := exec.Command("pdftotext", "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("pdftotext: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	headers = []string{"Date", "Merchant", "Amount"}
	all := [][]string{headers}
	for _, line := range strings.Split(stdout.String(), "\n") {
		m := meisaiLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		date, err := time.Parse("06/01/02", m[1])
		if err != nil {
			continue
		}
		merchant := strings.Join(strings.Fields(m[2]), " ")
		all = append(all, []string{date.Format("2006/01/02"), merchant, m[3]})
	}
	if len(all) == 1 {
		return nil, nil, fmt.Errorf("no transactions found in PDF")
	}
	return headers, all, nil
}
