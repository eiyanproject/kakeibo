package handlers

import (
	crand "crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kakeibo/internal/importer"
	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) ImportForm(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.Store.ListAccounts(r.Context(), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	web.Render(w, "import.html", map[string]any{"Accounts": accounts})
}

func tempUploadPath(token string) string {
	return filepath.Join(os.TempDir(), "kakeibo-upload-"+token+".csv")
}

func randomToken() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
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
		http.Error(w, "choose a CSV file", http.StatusBadRequest)
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

	f, err := os.Open(tempUploadPath(token))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	headers, rows, err := importer.ReadCSV(f)
	if err != nil {
		http.Error(w, "could not parse CSV: "+err.Error(), http.StatusBadRequest)
		return
	}

	profile, _ := h.Store.GetImportProfile(r.Context(), accountID)

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
		Invert:     r.FormValue("invert") == "on",
		HasHeader:  r.FormValue("has_header") == "on",
	}

	path := tempUploadPath(token)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "upload expired, please re-upload the file", http.StatusBadRequest)
		return
	}
	defer f.Close()
	defer os.Remove(path)

	headers, rows, err := importer.ReadCSV(f)
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
