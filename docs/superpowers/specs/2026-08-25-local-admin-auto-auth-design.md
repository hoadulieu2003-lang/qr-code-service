# Tự xác thực cho trang test QR local

## Mục tiêu

Rút ngắn kiểm thử thủ công: người dùng mở trang local, điền thông tin sản phẩm, bấm tạo QR và quét điện thoại mà không phải tìm hoặc nhập `API_KEY`.

## Phạm vi

Chỉ thay đổi hai route giao diện local:

- `GET /admin`
- `POST /admin/products`

Giữ nguyên `POST /api/products` và `GET /api/products/{trace_code}/qr.png`: chúng vẫn yêu cầu `X-API-Key` để hệ thống khác tích hợp an toàn.

## Thiết kế đã duyệt

`/admin` đã bị giới hạn ở request loopback (`127.0.0.1` hoặc `::1`). Vì vậy form local không hiển thị, nhận, lưu, hoặc kiểm tra trường `api_key`. Khi `POST /admin/products` đến từ loopback, server tạo sản phẩm qua cùng `createProduct` đã dùng cho API và dựa vào cấu hình server đã được kiểm tra lúc khởi động.

`LoadConfig` tiếp tục bắt buộc `API_KEY` không rỗng. Key này vẫn bảo vệ API tích hợp; việc bỏ ô form không làm server chạy với API mở.

Luồng mới:

```text
PC localhost: /admin -> điền Product -> POST /admin/products -> QR kết quả
Điện thoại cùng Wi-Fi: quét QR -> PUBLIC_BASE_URL/trace/{trace_code}
Hệ thống tích hợp: POST /api/products + X-API-Key -> giữ nguyên xác thực
```

## Trải nghiệm và lỗi

- Form chỉ có tám trường `Product`; không còn ô API key hoặc cảnh báo key sai.
- Form tiếp tục trả `400`, `409`, `503`, hoặc `500` cho lỗi dữ liệu, mã trùng, URL công khai chưa sẵn sàng, và lỗi nội bộ.
- Request từ LAN tới cả GET/POST `/admin` tiếp tục nhận `403` trước khi đọc form hoặc tạo dữ liệu.
- Người dùng local chỉ cần sửa hạn dùng để không trước ngày sản xuất, rồi bấm tạo QR.

## Kiểm thử nghiệm thu

1. `GET /admin` trên loopback trả form tám trường, không có input `api_key`.
2. POST form hợp lệ không có API key trả `201`, lưu sản phẩm, và render QR High/512 cho `trace_url`.
3. POST form có trường `api_key` tùy ý không dùng giá trị đó và không phản chiếu nó; xác thực form không phụ thuộc secret do người dùng gõ.
4. Request LAN tới `/admin` tiếp tục trả `403` và không ghi sản phẩm.
5. Test sai/mất `X-API-Key` cho JSON API tiếp tục trả `401`.
6. Sau khi service restart, người dùng local tạo một sản phẩm mới trong form và điện thoại mở đúng trang truy xuất sau khi quét QR.

## An toàn và giới hạn

Đây là tiện ích cho máy đang chạy server; bất kỳ process nào trên chính máy đó có thể mở `/admin`. Không đưa `/admin` qua reverse proxy, VPN, port-forwarding, hoặc Internet. Nếu cần màn quản trị mạng/public sau này, cần đăng nhập phiên quản trị, HTTPS và CSRF protection thay vì tự xác thực.
