package handlers

import (
	crand "crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"kakeibo/internal/importer"
	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

var (
	dateHeaderRe   = regexp.MustCompile(`(?i)date|日付`)
	amountHeaderRe = regexp.MustCompile(`(?i)amount|price|金額`)
)

// guessColumns picks which header holds the date/description/amount, preferring the
// account's saved mapping (if its columns still exist in this file), falling back to
// matching header names against common date/amount keywords, and finally to a
// positional guess so every file gets a usable mapping without asking the user.
func guessColumns(headers []string, prof *repo.ImportProfile) (dateCol, descCol, amountCol string) {
	has := func(name string) bool {
		for _, h := range headers {
			if h == name {
				return true
			}
		}
		return false
	}
	if prof != nil && has(prof.DateCol) && has(prof.DescCol) && has(prof.AmountCol) {
		return prof.DateCol, prof.DescCol, prof.AmountCol
	}

	for _, h := range headers {
		if dateCol == "" && dateHeaderRe.MatchString(h) {
			dateCol = h
		} else if amountCol == "" && amountHeaderRe.MatchString(h) {
			amountCol = h
		}
	}
	for _, h := range headers {
		if h != dateCol && h != amountCol {
			descCol = h
			break
		}
	}

	if dateCol == "" && len(headers) > 0 {
		dateCol = headers[0]
	}
	if amountCol == "" && len(headers) > 0 {
		amountCol = headers[len(headers)-1]
	}
	if descCol == "" {
		for _, h := range headers {
			if h != dateCol && h != amountCol {
				descCol = h
				break
			}
		}
	}
	return dateCol, descCol, amountCol
}

func (h *Handlers) ImportForm(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.Store.ListAccounts(r.Context(), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	web.Render(w, "import.html", map[string]any{"Accounts": accounts})
}

func tempUploadPath(token string) string {
	return filepath.Join(os.TempDir(), "kakeibo-upload-"+token)
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}

// readImportFile dispatches to the PDF or CSV reader based on the file's actual content
// (not its extension or declared content-type), so uploads are read strictly by what they are.
func readImportFile(path string) (headers []string, rows [][]string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	magic := make([]byte, 5)
	n, _ := f.Read(magic)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}
	if n >= 5 && string(magic) == "%PDF-" {
		return importer.ReadPDF(f)
	}
	return importer.ReadCSV(f)
}

func (h *Handlers) ImportPreview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "bad upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	accountID, err := strconv.ParseInt(r.FormValue("account_id"), 10, 64)
	if err != nil {
		http.Error(w, "select an account", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "choose a CSV or PDF file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	token := randomToken()
	dst, err := os.Create(tempUploadPath(token))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dst.Close()

	headers, rows, err := readImportFile(tempUploadPath(token))
	if err != nil {
		http.Error(w, "could not parse file: "+err.Error(), http.StatusBadRequest)
		return
	}

	profile, _ := h.Store.GetImportProfile(r.Context(), accountID)
	dateCol, descCol, amountCol := guessColumns(headers, profile)

	preview := rows
	if len(preview) > 6 {
		preview = preview[:6]
	}

	web.Render(w, "import-preview", map[string]any{
		"AccountID":   accountID,
		"Token":       token,
		"Filename":    header.Filename,
		"Headers":     headers,
		"PreviewRows": preview,
		"DateCol":     dateCol,
		"DescCol":     descCol,
		"AmountCol":   amountCol,
		"Profile":     profile,
	})
}

func (h *Handlers) ImportCommit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	accountID, err := strconv.ParseInt(r.FormValue("account_id"), 10, 64)
	if err != nil {
		http.Error(w, "bad account", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	filename := r.FormValue("filename")
	mapping := importer.Mapping{
		DateCol:    r.FormValue("date_col"),
		DescCol:    r.FormValue("desc_col"),
		AmountCol:  r.FormValue("amount_col"),
		DateLayout: r.FormValue("date_layout"),
		// This app only tracks spending, and every supported source (CSV exports,
		// meisai PDFs) represents a purchase as a positive number — always flip the
		// sign so spend is stored negative, matching the app's convention.
		Invert:    true,
		HasHeader: r.FormValue("has_header") == "on",
	}

	path := tempUploadPath(token)
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "upload expired, please re-upload the file", http.StatusBadRequest)
		return
	}
	defer os.Remove(path)

	headers, rows, err := readImportFile(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parsed, err := importer.ParseRows(headers, rows, mapping)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rules, err := h.Store.ListRules(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uncategorized, err := h.Store.UncategorizedID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	batchID, err := h.Store.CreateImportBatch(ctx, accountID, filename, len(parsed))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	inserted, skipped := 0, 0
	for _, row := range parsed {
		catID := uncategorized
		if id, ok := repo.MatchCategory(rules, row.Merchant); ok {
			catID = id
		}
		hash := importer.DedupHash(row.Date, row.Merchant, row.AmountMinor)
		bID := batchID
		_, ok, err := h.Store.InsertTransaction(ctx, repo.Transaction{
			AccountID:          accountID,
			TxnDate:            row.Date,
			MerchantRaw:        row.Merchant,
			MerchantNormalized: strings.ToUpper(row.Merchant),
			AmountMinor:        row.AmountMinor,
			CategoryID:         &catID,
			ImportBatchID:      &bID,
			DedupHash:          hash,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			inserted++
		} else {
			skipped++
		}
	}

	_ = h.Store.SaveImportProfile(ctx, repo.ImportProfile{
		AccountID: accountID, DateCol: mapping.DateCol, DescCol: mapping.DescCol, AmountCol: mapping.AmountCol,
		DateLayout: mapping.DateLayout, InvertAmount: mapping.Invert, HasHeader: mapping.HasHeader,
	})

	web.Render(w, "import-result", map[string]any{
		"Inserted":  inserted,
		"Skipped":   skipped,
		"AccountID": accountID,
	})
}
