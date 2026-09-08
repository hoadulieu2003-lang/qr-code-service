# Checklist bàn giao module QR truy xuất nguồn gốc

Điền ngày, người kiểm tra và kết quả khi bàn giao. Mỗi ô chỉ được tích sau khi có bằng chứng (output command, ảnh QR hoặc quét điện thoại thực tế).

## Source và build

- [ ] Source nằm nguyên ở thư mục gốc; không di chuyển các file Go.
- [ ] `go test ./...` trả exit code `0`.
- [ ] `go build -o bond-verify.exe .` trả exit code `0`.
- [ ] `git diff --check` không có lỗi whitespace.
- [ ] `scripts/start-local.ps1`, `scripts/check-health.ps1`, `scripts/test-api.ps1` có trong source bàn giao.
- [ ] `docs/handover/` có đủ 4 tài liệu hướng dẫn này.

## Vận hành cục bộ

- [ ] Đã tạo `.env` từ `sample.env`, cấu hình `PORT`, `API_KEY`, `PUBLIC_BASE_URL` và `DATA_FILE`.
- [ ] `PUBLIC_BASE_URL` là URL HTTP/HTTPS tuyệt đối, không có dấu `/` ở cuối; khi test điện thoại dùng IPv4 LAN của máy server.
- [ ] Start script chạy foreground; có thể dừng bằng `Ctrl+C`.
- [ ] `.\scripts\check-health.ps1 -Port <port>` báo `PASS`.
- [ ] `http://localhost:<port>/admin` hiển thị form và không yêu cầu API key.
- [ ] `http://<IPv4-LAN>:<port>/admin` trả `403`.

## API và QR

- [ ] `POST /api/products` thiếu/sai `X-API-Key` trả `401`.
- [ ] Tạo sản phẩm hợp lệ trả `201` với `trace_code`, `trace_url`, `qr_url`.
- [ ] Tạo lại cùng `trace_code` trả `409`; sản phẩm đầu không bị thay thế.
- [ ] `GET /api/products/{traceCode}/qr.png` với `X-API-Key` trả `200 image/png`.
- [ ] `GET /trace/{traceCode}` không cần API key và trả `200`.
- [ ] QR mã hóa đúng `trace_url`, không chứa API key.
- [ ] Điện thoại cùng Wi-Fi quét QR thật và mở được trang truy xuất.
- [ ] Trang điện thoại hiện đúng sáu nhóm thông tin sản phẩm và tiếng Việt không lỗi.
- [ ] Hoàn thành cả 10 kịch bản trong [hướng dẫn kiểm thử](03-test-guide.md).

## Dữ liệu và bí mật

- [ ] `.env` không bị commit, gửi qua chat, ghi vào log hoặc đưa vào biên bản.
- [ ] `data/products.json` đã được backup trước khi nâng cấp/đổi máy.
- [ ] `data/`, `.env` và các file `.exe` là runtime artifacts bị Git bỏ qua.
- [ ] Hệ thống gọi API lưu `trace_url` được trả về, không tự ghép URL.

## Điều kiện trước khi mở cho Internet

Triển khai local/LAN trong tài liệu này **không phải** cấu hình public production. Trước khi mở Internet, cần tối thiểu:

- [ ] HTTPS hợp lệ ở reverse proxy hoặc ngay trong dịch vụ; không phát QR HTTP công khai.
- [ ] Cơ chế đăng nhập phiên quản trị thực sự cho admin, thay cho form loopback phục vụ test.
- [ ] Bảo vệ CSRF cho các form ghi dữ liệu.
- [ ] Database bền vững, migration và backup/restore được kiểm tra; không chỉ dựa vào `data/products.json` cục bộ.
- [ ] Quản lý API key bằng secret store/biến môi trường, rotation và phân quyền caller.
- [ ] Giới hạn truy cập, logging/monitoring không làm rò rỉ dữ liệu/bí mật, và chính sách firewall phù hợp.
