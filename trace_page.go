package main

import (
	"errors"
	"html/template"
	"net/http"

	"github.com/go-chi/chi"
)

type traceView struct {
	Name               string
	ProductAndBatch    string
	ManufacturedAt     string
	ExpiresAt          string
	Origin             string
	VerificationStatus string
}

var traceTemplate = template.Must(template.New("trace").Parse(`<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Truy xuất nguồn gốc</title>
  <style>
    :root { color-scheme: light; font-family: Arial, sans-serif; color: #172033; background: #f4f7fb; }
    body { margin: 0; padding: 24px 16px; }
    main { max-width: 560px; margin: 0 auto; background: #fff; border-radius: 16px; box-shadow: 0 8px 24px #1720331a; overflow: hidden; }
    header { background: #0f766e; color: #fff; padding: 24px; }
    h1 { margin: 0; font-size: 1.35rem; }
    dl { margin: 0; padding: 8px 24px 24px; }
    div { padding: 16px 0; border-bottom: 1px solid #e5e7eb; }
    div:last-child { border-bottom: 0; }
    dt { color: #526071; font-size: .82rem; font-weight: 700; margin-bottom: 5px; text-transform: uppercase; }
    dd { margin: 0; font-size: 1rem; line-height: 1.45; word-break: break-word; }
    .verified { color: #047857; font-weight: 700; }
  </style>
</head>
<body>
  <main>
    <header><h1>Truy xuất nguồn gốc</h1></header>
    <dl>
      <div><dt>Tên sản phẩm</dt><dd>{{.Name}}</dd></div>
      <div><dt>Mã sản phẩm / lô</dt><dd>{{.ProductAndBatch}}</dd></div>
      <div><dt>Ngày sản xuất</dt><dd>{{.ManufacturedAt}}</dd></div>
      <div><dt>Hạn dùng</dt><dd>{{.ExpiresAt}}</dd></div>
      <div><dt>Nơi sản xuất</dt><dd>{{.Origin}}</dd></div>
      <div><dt>Trạng thái xác thực</dt><dd class="verified">{{.VerificationStatus}}</dd></div>
    </dl>
  </main>
</body>
</html>`))

var traceNotFoundTemplate = template.Must(template.New("not-found").Parse(`<!doctype html>
<html lang="vi"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Không tìm thấy sản phẩm</title></head>
<body><main><h1>Không tìm thấy sản phẩm</h1><p>Mã truy xuất không tồn tại hoặc đã bị gỡ.</p></main></body></html>`))

func publicTracePageHandler(store *ProductStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		product, err := store.Get(chi.URLParam(request, "traceCode"))
		if errors.Is(err, ErrProductNotFound) {
			writeTraceNotFound(writer)
			return
		}
		if err != nil {
			writeTraceNotFound(writer)
			return
		}

		view := traceView{
			Name:               product.Name,
			ProductAndBatch:    product.ProductCode + " / " + product.BatchCode,
			ManufacturedAt:     product.ManufacturedAt,
			ExpiresAt:          product.ExpiresAt,
			Origin:             product.Origin,
			VerificationStatus: product.VerificationStatus,
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_ = traceTemplate.Execute(writer, view)
	}
}

func writeTraceNotFound(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusNotFound)
	_ = traceNotFoundTemplate.Execute(writer, nil)
}
