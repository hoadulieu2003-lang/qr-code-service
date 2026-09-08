package main

import (
	"encoding/base64"
	"errors"
	"html/template"
	"net"
	"net/http"
)

type adminView struct {
	Product  Product
	Error    string
	TraceURL string
	QRBase64 string
}

var adminTemplate = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tạo mã QR truy xuất</title>
  <style>
    :root { color-scheme: light; font-family: Arial, sans-serif; color: #172033; background: #f4f7fb; }
    body { margin: 0; padding: 24px 16px; }
    main { max-width: 720px; margin: 0 auto; background: #fff; border-radius: 16px; box-shadow: 0 8px 24px #1720331a; overflow: hidden; }
    header { background: #0f766e; color: #fff; padding: 24px; }
    h1 { margin: 0; font-size: 1.4rem; }
    header p { margin: 8px 0 0; line-height: 1.45; }
    form, .result { padding: 24px; }
    .fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
    label { display: grid; gap: 6px; color: #334155; font-size: .9rem; font-weight: 700; }
    label.full { grid-column: 1 / -1; }
    input, select, button { box-sizing: border-box; border-radius: 8px; font: inherit; }
    input, select { width: 100%; border: 1px solid #cbd5e1; padding: 10px 12px; }
    button { margin-top: 20px; border: 0; background: #0f766e; color: #fff; cursor: pointer; font-weight: 700; padding: 12px 16px; }
    .error { margin: 0 24px; border-radius: 8px; background: #fef2f2; color: #991b1b; padding: 12px 16px; }
    .result { border-top: 1px solid #e2e8f0; text-align: center; }
    .result h2 { margin: 0 0 8px; }
    .result p { overflow-wrap: anywhere; }
    img { display: block; width: min(100%, 512px); height: auto; margin: 20px auto 0; }
    @media (max-width: 560px) { .fields { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>Tạo mã QR truy xuất nguồn gốc</h1>
      <p>Nhập sản phẩm, tạo QR rồi dùng điện thoại cùng Wi-Fi để quét kiểm tra.</p>
    </header>
    {{if .Error}}<p class="error" role="alert">{{.Error}}</p>{{end}}
    <form method="post" action="/admin/products">
      <div class="fields">
        <label>Mã truy xuất<input name="trace_code" required value="{{.Product.TraceCode}}"></label>
        <label>Mã sản phẩm<input name="product_code" required value="{{.Product.ProductCode}}"></label>
        <label class="full">Tên sản phẩm<input name="name" required value="{{.Product.Name}}"></label>
        <label>Mã lô<input name="batch_code" required value="{{.Product.BatchCode}}"></label>
        <label>Nơi sản xuất<input name="origin" required value="{{.Product.Origin}}"></label>
        <label>Ngày sản xuất<input type="date" name="manufactured_at" required value="{{.Product.ManufacturedAt}}"></label>
        <label>Hạn dùng<input type="date" name="expires_at" required value="{{.Product.ExpiresAt}}"></label>
        <label>Trạng thái xác thực
          <select name="verification_status" required>
            <option value="verified" {{if ne .Product.VerificationStatus "unverified"}}selected{{end}}>verified</option>
            <option value="unverified" {{if eq .Product.VerificationStatus "unverified"}}selected{{end}}>unverified</option>
          </select>
        </label>
      </div>
      <button type="submit">Tạo QR</button>
    </form>
    {{if .TraceURL}}
      <section class="result" aria-live="polite">
        <h2>Đã tạo QR thành công</h2>
        <p>Quét mã bên dưới để mở trang truy xuất:</p>
        <p><a href="{{.TraceURL}}">{{.TraceURL}}</a></p>
        <img src="data:image/png;base64,{{.QRBase64}}" alt="Mã QR truy xuất nguồn gốc">
      </section>
    {{end}}
  </main>
</body>
</html>`))

func renderAdmin(writer http.ResponseWriter, status int, view adminView) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	_ = adminTemplate.Execute(writer, view)
}

func isLoopbackRequest(request *http.Request) bool {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func writeAdminLocalOnly(writer http.ResponseWriter) {
	http.Error(writer, "Trang quan tri chi cho phep truy cap tu localhost.", http.StatusForbidden)
}

func adminFormHandler() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !isLoopbackRequest(request) {
			writeAdminLocalOnly(writer)
			return
		}
		renderAdmin(writer, http.StatusOK, adminView{})
	}
}

func adminCreateProductHandler(config Config, store *ProductStore) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !isLoopbackRequest(request) {
			writeAdminLocalOnly(writer)
			return
		}
		if err := request.ParseForm(); err != nil {
			renderAdmin(writer, http.StatusBadRequest, adminView{Error: "Dữ liệu biểu mẫu không hợp lệ."})
			return
		}

		product := Product{
			TraceCode:          request.PostForm.Get("trace_code"),
			ProductCode:        request.PostForm.Get("product_code"),
			Name:               request.PostForm.Get("name"),
			BatchCode:          request.PostForm.Get("batch_code"),
			ManufacturedAt:     request.PostForm.Get("manufactured_at"),
			ExpiresAt:          request.PostForm.Get("expires_at"),
			Origin:             request.PostForm.Get("origin"),
			VerificationStatus: request.PostForm.Get("verification_status"),
		}
		result, err := createProduct(config, store, product)
		if err != nil {
			status := http.StatusInternalServerError
			message := "Không thể lưu sản phẩm. Hãy thử lại."
			switch {
			case errors.Is(err, ErrInvalidProduct):
				status = http.StatusBadRequest
				message = err.Error()
			case errors.Is(err, ErrPublicTraceURLUnavailable):
				status = http.StatusServiceUnavailable
				message = "URL truy xuất công khai chưa sẵn sàng."
			case errors.Is(err, ErrDuplicateTraceCode):
				status = http.StatusConflict
				message = "Mã truy xuất đã tồn tại."
			}
			renderAdmin(writer, status, adminView{Product: product, Error: message})
			return
		}

		renderAdmin(writer, http.StatusCreated, adminView{
			Product:  product,
			TraceURL: result.TraceURL,
			QRBase64: base64.StdEncoding.EncodeToString(result.qrPNG),
		})
	}
}
