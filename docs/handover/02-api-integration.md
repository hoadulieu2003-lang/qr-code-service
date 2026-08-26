# Tích hợp API tạo QR truy xuất nguồn gốc

API này dành cho hệ thống quản lý sản phẩm gọi từ server hoặc backend đáng tin cậy. Mỗi lần tạo thành công, module lưu sản phẩm, sinh QR mã hóa `trace_url`, và trả lại các URL để hệ thống gọi lưu cùng sản phẩm của mình.

## Bảo mật và URL cơ sở

Các API quản trị yêu cầu header `X-API-Key`. Không đặt API key trong app mobile, JavaScript chạy trên trình duyệt, QR, hoặc URL. Người dùng cuối chỉ truy cập được trang công khai `/trace/{traceCode}`.

Trong các ví dụ, thay hai biến bằng giá trị của môi trường chạy:

```powershell
$baseUrl = 'http://192.168.1.22:18080'
$apiKey = Read-Host 'Nhap API key'
```

Khi tích hợp thực tế, lấy `baseUrl` và key từ biến môi trường hoặc kho bí mật của backend, không hard-code vào source.

## Dữ liệu sản phẩm: đủ 8 trường

| JSON field | Bắt buộc | Quy tắc |
| --- | --- | --- |
| `trace_code` | Có | Duy nhất; chỉ chữ, số, dấu `-`, `_`; hệ thống có trim khoảng trắng đầu/cuối |
| `product_code` | Có | Mã sản phẩm, không rỗng |
| `name` | Có | Tên sản phẩm, không rỗng; hỗ trợ tiếng Việt/Unicode |
| `batch_code` | Có | Mã lô, không rỗng |
| `manufactured_at` | Có | Ngày `YYYY-MM-DD` |
| `expires_at` | Có | Ngày `YYYY-MM-DD`, không trước `manufactured_at` |
| `origin` | Có | Xuất xứ, không rỗng |
| `verification_status` | Có | Chỉ `verified` hoặc `unverified` |

Ví dụ dữ liệu:

```json
{
  "trace_code": "SP-CA-PHE-0001",
  "product_code": "CP-250G",
  "name": "Cà phê rang xay 250 g",
  "batch_code": "LO-2026-001",
  "manufactured_at": "2026-08-26",
  "expires_at": "2027-08-26",
  "origin": "Đắk Lắk, Việt Nam",
  "verification_status": "verified"
}
```

## Các endpoint

| Phương thức và đường dẫn | Xác thực | Kết quả chính |
| --- | --- | --- |
| `POST /api/products` | `X-API-Key` | Tạo sản phẩm và trả QR/trace URL |
| `GET /api/products/{traceCode}/qr.png` | `X-API-Key` | Tải PNG QR 512 × 512 |
| `GET /trace/{traceCode}` | Không | Trang HTML truy xuất công khai cho điện thoại |
| `GET /health` | Không | `200 OK` nếu dịch vụ đang sống |

### 1. Tạo sản phẩm và QR

```powershell
$product = @{
  trace_code = 'SP-CA-PHE-0001'
  product_code = 'CP-250G'
  name = 'Cà phê rang xay 250 g'
  batch_code = 'LO-2026-001'
  manufactured_at = '2026-08-26'
  expires_at = '2027-08-26'
  origin = 'Đắk Lắk, Việt Nam'
  verification_status = 'verified'
} | ConvertTo-Json

$created = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/products" `
  -Headers @{ 'X-API-Key' = $apiKey } `
  -ContentType 'application/json; charset=utf-8' `
  -Body $product

$created
```

Trả về `201 Created`:

```json
{
  "trace_code": "SP-CA-PHE-0001",
  "trace_url": "http://192.168.1.22:18080/trace/SP-CA-PHE-0001",
  "qr_url": "http://192.168.1.22:18080/api/products/SP-CA-PHE-0001/qr.png"
}
```

**Quy tắc tích hợp:** lưu chính xác `trace_code` và `trace_url` mà API trả về. Không tự ghép URL ở hệ thống gọi, vì QR được tạo từ `trace_url` do module quyết định. `qr_url` là đường dẫn tải ảnh cho hệ thống quản trị, vẫn phải gọi kèm `X-API-Key`.

Ví dụ cURL (Windows dùng `curl.exe` để tránh alias PowerShell):

```powershell
curl.exe -X POST "$baseUrl/api/products" `
  -H "X-API-Key: $apiKey" `
  -H 'Content-Type: application/json' `
  --data '{"trace_code":"SP-CA-PHE-0001","product_code":"CP-250G","name":"Ca phe rang xay 250 g","batch_code":"LO-2026-001","manufactured_at":"2026-08-26","expires_at":"2027-08-26","origin":"Dak Lak, Viet Nam","verification_status":"verified"}'
```

### 2. Tải ảnh QR PNG

```powershell
Invoke-WebRequest -Uri $created.qr_url `
  -Headers @{ 'X-API-Key' = $apiKey } `
  -OutFile '.\SP-CA-PHE-0001.png'
```

```powershell
curl.exe -f -H "X-API-Key: $apiKey" `
  -o .\SP-CA-PHE-0001.png `
  "$baseUrl/api/products/SP-CA-PHE-0001/qr.png"
```

Ảnh là `image/png`, kích thước 512 × 512 và QR chứa `trace_url`. Không cần gọi endpoint QR nếu chỉ cần mở trang truy xuất: dùng trực tiếp `trace_url`.

### 3. Mở trang truy xuất công khai

```powershell
Invoke-WebRequest -UseBasicParsing "$baseUrl/trace/SP-CA-PHE-0001"
```

Không thêm API key. Đây là URL mà QR dẫn điện thoại đến.

### 4. Health check

```powershell
Invoke-WebRequest -UseBasicParsing "$baseUrl/health"
```

## Mã trạng thái và cách xử lý

| HTTP | Khi nào xảy ra | Hành động caller |
| --- | --- | --- |
| `201` | Tạo mới thành công | Lưu `trace_code`, `trace_url`, `qr_url`; có thể tải QR |
| `400` | JSON, content type hoặc 8 trường không hợp lệ | Sửa dữ liệu, không retry nguyên trạng |
| `401` | Thiếu/sai `X-API-Key` | Kiểm tra secret phía server, tuyệt đối không log key |
| `404` | QR hoặc trace code không tồn tại | Kiểm tra mã đã lưu/còn dữ liệu hay không |
| `409` | `trace_code` đã tồn tại | Không coi là tạo thành công; dùng mã mới hoặc tra dữ liệu cũ |
| `500` | Lưu dữ liệu hoặc tạo QR thất bại | Không tự giả định QR đã tạo; kiểm tra `/trace/{traceCode}` trước khi retry |
| `503` | `PUBLIC_BASE_URL` không dùng được để sinh URL công khai | Sửa cấu hình URL tuyệt đối HTTP/HTTPS, không có `/` ở cuối |

Mọi lỗi API hiện trả JSON dạng `{ "error": "..." }`. Các lỗi validation giữ thông điệp chi tiết để backend hiển thị/log nội bộ an toàn.

## Script kiểm thử nhanh cho backend

Thay vì chép API key vào lệnh, chạy script đã có. Script đọc `.env` nội bộ, tạo một sản phẩm, kiểm tra trang trace trả `200`, và chỉ in ba URL/mã an toàn:

```powershell
.\scripts\test-api.ps1 -TraceCode 'SP-API-KIEM-THU-001'
```

Nếu mã đã tồn tại, script dừng và báo rõ `409`; không ghi đè sản phẩm cũ.
