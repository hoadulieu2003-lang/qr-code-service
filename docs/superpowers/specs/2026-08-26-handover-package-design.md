# Gói bàn giao module QR truy xuất nguồn gốc

## Mục tiêu

Biến worktree hiện tại thành một gói có thể bàn giao: người nhận biết file nào là source, cách cấu hình/chạy/dừng, cách test giao diện và điện thoại, và cách hệ thống khác gọi API tạo sản phẩm/QR.

## Quy tắc sắp xếp

- Giữ nguyên toàn bộ file Go ở root (`main.go`, `app.go`, `handlers.go`, các test) để không làm thay đổi lệnh `go test ./...` hoặc `go build .` đang hoạt động.
- Giữ `docs/superpowers/` làm lịch sử thiết kế/kế hoạch kỹ thuật; không phải tài liệu vận hành bàn giao.
- Thêm một nhánh tài liệu rõ ràng tại `docs/handover/` và script Windows tại `scripts/`.
- Không commit/copy secret hoặc dữ liệu vận hành: `.env`, `data/`, `*.exe` tiếp tục là file runtime bị ignore.

## Cấu trúc bàn giao

```text
README.md                         # Lối vào: tổng quan, link đến tài liệu bàn giao
sample.env                        # Mẫu cấu hình không chứa secret
scripts/
  start-local.ps1                 # Kiểm tra .env, build và chạy foreground
  check-health.ps1                # Kiểm tra /health và form local /admin
  test-api.ps1                    # Tạo sản phẩm qua API với X-API-Key lấy từ .env
docs/handover/
  01-quick-start.md               # Yêu cầu, cấu hình, chạy/dừng, URL sử dụng
  02-api-integration.md           # Hợp đồng API, payload, response, status, PowerShell/cURL
  03-test-guide.md                # UI, QR điện thoại, 10 case runtime, API regression
  04-handover-checklist.md        # Checklist nghiệm thu và phạm vi/giới hạn production
```

## Script và hợp đồng vận hành

### start-local.ps1

Chạy từ project root. Script phải:

1. Từ chối nếu `.env` không tồn tại.
2. Từ chối nếu không tìm thấy `go` trong `PATH`.
3. Build `bond.exe` bằng `go build -o bond.exe .`.
4. Chạy `bond.exe` foreground để người vận hành thấy log và dừng bằng `Ctrl+C`.

Script không được in `API_KEY`, copy `.env`, hay chạy nền. Lựa chọn foreground giúp tránh hiểu lầm service vẫn hoạt động sau khi terminal đóng.

### check-health.ps1

Nhận `-Port` (mặc định `18080`), gọi `GET /health` và `GET /admin` qua `localhost`. Thành công khi cả hai trả `200`, form chứa trường `trace_code`, và không có trường `api_key`. Nếu fail, script trả exit code khác 0 cùng hướng dẫn URL/port.

### test-api.ps1

Nhận `-TraceCode` bắt buộc và các tham số product có default demo an toàn. Đọc `API_KEY`, `PUBLIC_BASE_URL`, và `PORT` từ `.env` mà không in key. Gọi `POST /api/products` với header `X-API-Key`, in response gồm `trace_code`, `trace_url`, `qr_url`, rồi kiểm tra URL truy xuất trả HTML. Mã trùng được báo rõ `409` và không bị script ghi đè.

## Nội dung tài liệu

### 01-quick-start.md

Nêu Go requirement, Wi-Fi/LAN requirement cho điện thoại, copy `sample.env` sang `.env`, cách đặt `PUBLIC_BASE_URL` bằng IPv4 LAN, lệnh `scripts/start-local.ps1`, URL `http://localhost:<PORT>/admin`, và cách dừng server. Phải giải thích QR điện thoại không dùng `localhost` hay `127.0.0.1`.

### 02-api-integration.md

Mô tả rõ hai bề mặt:

- Form `localhost /admin`: chỉ cho loopback, tự xác thực nội bộ, dùng để test nhanh; không phải API tích hợp.
- API: `POST /api/products`, `GET /api/products/{trace_code}/qr.png`, `GET /trace/{trace_code}`, `GET /health`; yêu cầu header/key ở endpoint quản trị.

Phải có JSON Product đầy đủ tám trường, JSON response, bảng status `201`, `400`, `401`, `404`, `409`, `500`, `503`, ví dụ PowerShell và curl, và nguyên tắc client lưu `trace_url` thay vì tự ghép URL.

### 03-test-guide.md

Ghi test trước giao diện (không có API key), date rule, QR 512px, quét điện thoại cùng Wi-Fi, và expected six trace fields. Ghi 10 kịch bản runtime đã chạy ngày 26-08-2026: bảy QR hợp lệ, một mã trùng, hai rejection cases (date invalid/QR capacity) với trạng thái mong đợi. Nêu rõ test server không thay thế xác nhận quét camera điện thoại thật.

### 04-handover-checklist.md

Cho checklist trước bàn giao: test pass, health 200, form localhost 200/no key, LAN `/admin` 403, một QR quét bằng điện thoại, API 401/201/409, backup `data/products.json`, secret quản lý qua `.env`, và giới hạn khi production (HTTPS, session admin, CSRF, database/backup policy).

## Kiểm thử nghiệm thu gói bàn giao

1. `go test ./...` và `go build -o bond-verify.exe .` pass.
2. Mọi link tài liệu từ README tồn tại; không link nhầm tới secret hoặc runtime data.
3. `start-local.ps1` nêu rõ foreground lifecycle và `check-health.ps1` pass với service đang chạy.
4. `test-api.ps1` tạo được một mã mới qua API, kiểm tra trace page, và không hiển thị API key.
5. `git status --ignored --short` vẫn cho `.env`, `data/`, executables là ignored; không có secret/data thật trong staged changes.

## Ngoài phạm vi

Không refactor package Go, không thêm database/tenant/account, không deploy Internet, không commit `.env`/data thật/binary, và không tuyên bố tuân thủ pháp lý truy xuất nguồn gốc. Đây là gói bàn giao kỹ thuật cho MVP local/LAN.
