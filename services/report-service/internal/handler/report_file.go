package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

const (
	headerContentType        = "Content-Type"
	headerContentDisposition = "Content-Disposition"
	headerContentLength      = "Content-Length"
	headerCacheControl       = "Cache-Control"
	privateNoStore           = "private, no-store"
	asciiFallbackRune        = '_'
	dispositionFormat        = `attachment; filename="%s"; filename*=UTF-8''%s`
)

func writeFile(response http.ResponseWriter, file domain.ReportFile) {
	header := response.Header()
	header.Set(headerContentType, file.ContentType)
	header.Set(headerContentDisposition, contentDisposition(file.Name))
	header.Set(headerContentLength, strconv.Itoa(len(file.Content)))
	header.Set(headerCacheControl, privateNoStore)
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(file.Content)
}

func contentDisposition(name string) string {
	fallback := strings.Map(func(character rune) rune {
		if character < unicode.MaxASCII && (unicode.IsLetter(character) || unicode.IsDigit(character) ||
			strings.ContainsRune("-._", character)) {
			return character
		}

		return asciiFallbackRune
	}, name)

	return fmt.Sprintf(dispositionFormat, fallback, url.PathEscape(name))
}
