# Trang nội bộ tạo sản phẩm và QR để kiểm thử bằng điện thoại

## Mục tiêu

Cho người kiểm thử tự nhập một sản phẩm bất kỳ từ trình duyệt trên máy đang chạy dịch vụ, tạo bản ghi truy xuất, và thấy QR ngay trên trang để quét bằng điện thoại. Đây là lớp giao diện tiện dụng cho MVP hiện có, không thay thế các API tích hợp.

## Phạm vi đã chốt

Bao gồm:

- Trang form nội bộ tại `GET /admin`.
- Gửi form đến `POST /admin/products` để tạo một sản phẩm.
- Hiển thị QR, mã truy xuất và URL truy xuất sau khi tạo thành công.
- Xác thực bằng API key nhập trong form, tái dùng cùng nguyên tắc kiểm tra với API.
- Kiểm thử tự động cho luồng thành công và lỗi chính.

Không bao gồm đăng nhập/tài khoản, danh sách sản phẩm, sửa/xóa, analytics, upload dữ liệu hàng loạt, hay đưa giao diện ra Internet.

## Quyết định kiến trúc

Chọn form HTML do Go render ở phía server. Không dùng JavaScript gọi API quản trị, vì cách đó buộc trình duyệt phải tự xử lý API key và dễ làm lộ khóa trong mã nguồn hoặc công cụ trình duyệt.

Luồng tạo sản phẩm:

```text
Máy tính: GET /admin -> nhập Product + API key -> POST /admin/products
                                                     |
                                                     v
                                      xác thực + cùng logic tạo sản phẩm
                                                     |
                                                     v
                                  products.json + QR encode trace_url
                                                     |
                                                     v
                                    HTML kết quả chứa QR dạng data URI

Điện thoại: quét QR -> GET {PUBLIC_BASE_URL}/trace/{trace_code}
```

QR được embed dưới dạng PNG base64 trong trang kết quả. Thẻ ảnh trình duyệt không thể tự gửi `X-API-Key` cho endpoint QR hiện hữu, vì vậy embed QR giúp màn hình tạo QR hoạt động mà không mở công khai endpoint quản trị.

## Trải nghiệm sử dụng

Người kiểm thử mở `http://localhost:18080/admin` trên chính máy chạy service. Form gồm:

- `trace_code`, `product_code`, `name`, `batch_code`, `origin` là ô văn bản.
- `manufactured_at`, `expires_at` là ô ngày.
- `verification_status` là lựa chọn `verified` hoặc `unverified`.
- API key là ô `password`, bắt buộc và không được tự điền/lưu lại trên trang kết quả.

Sau khi bấm tạo:

- Nếu thành công, trang hiển thị thông báo tạo thành công, QR 512 px, `trace_code`, và `trace_url` để đối chiếu. Người dùng dùng camera điện thoại quét mã trên màn hình.
- Nếu lỗi, trang báo lỗi tiếng Việt dễ hiểu và giữ lại mọi trường sản phẩm đã nhập. Trường API key luôn để trống khi render lại.

QR luôn chứa `Config.TraceURL(trace_code)`, tức URL xây từ `PUBLIC_BASE_URL`. Với kiểm thử điện thoại cùng Wi-Fi, cấu hình hiện có phải là địa chỉ IPv4 LAN, không phải `localhost` hay `127.0.0.1`.

## Route, trách nhiệm và tái sử dụng

| Route / thành phần | Trách nhiệm |
| --- | --- |
| `GET /admin` | Render form rỗng. Không tiết lộ API key hoặc dữ liệu tồn tại. |
| `POST /admin/products` | Đọc `application/x-www-form-urlencoded`, kiểm tra API key, chuẩn hóa/xác thực `Product`, lưu và tạo QR kết quả. |
| Logic tạo sản phẩm dùng chung | Tạo `trace_url`, gọi `ProductStore.Create`, và phân biệt lỗi validation/trùng/lỗi lưu. Cả API JSON lẫn form HTML dùng chung để hành vi không bị lệch. |
| Template trang admin | Render form và kết quả qua `html/template`, escape mọi nội dung sản phẩm. |
| `POST /api/products` và QR PNG API | Giữ nguyên hợp đồng JSON, header `X-API-Key`, và endpoint hiện tại. |

Các file dự kiến được chạm khi triển khai là `app.go` (đăng ký route), `handlers.go` hoặc một lớp service nhỏ (logic tạo dùng chung), một file trang admin mới, và test Go tương ứng. Dữ liệu thật, `.env`, `data/products.json`, và API hợp đồng hiện có không bị đổi ngoài sản phẩm do người dùng tự tạo lúc chạy thử.

## Xác thực và xử lý lỗi

- Form kiểm tra API key bằng hàm so sánh thời gian hằng đang dùng cho API. API key không đi vào URL, cookie, local storage, log, tệp JSON, hay HTML phản hồi.
- Form được mở qua `localhost` trên máy tính để API key không rời máy trong bài test. Service vẫn có thể trả `trace_url` IP LAN cho điện thoại.
- `400 Bad Request`: trường thiếu/sai định dạng, ngày sai, hạn dùng trước ngày sản xuất, hoặc trạng thái không hợp lệ.
- `401 Unauthorized`: API key thiếu hoặc sai.
- `409 Conflict`: `trace_code` đã tồn tại.
- `503 Service Unavailable`: `PUBLIC_BASE_URL` chưa hợp lệ nên không thể tạo QR điện thoại quét được.
- Lỗi nội bộ khi đọc/ghi/làm QR trả `500` với thông báo tổng quát, không lộ secret hay đường dẫn hệ thống.
- Vì đây là công cụ local qua HTTP, trước khi mở ra Internet phải bổ sung HTTPS, xác thực phiên quản trị và CSRF protection.

## Kiểm thử và nghiệm thu

Các test Go phải kiểm tra:

1. `GET /admin` trả HTML form với đủ trường sản phẩm và API key password input.
2. Form hợp lệ với API key đúng trả thành công, ghi sản phẩm, hiển thị QR PNG data URI và `trace_url` đúng theo `PUBLIC_BASE_URL`.
3. QR được tạo encode đúng `trace_url` (kiểm tra bằng decode QR độc lập hoặc assertion tương đương của luồng tạo QR).
4. API key sai trả `401`, không ghi sản phẩm và không render lại giá trị key.
5. Dữ liệu không hợp lệ trả `400`, mã truy xuất trùng trả `409`, và thiếu/sai `PUBLIC_BASE_URL` trả `503`.
6. Test hiện có của `POST /api/products`, `GET /api/products/{trace_code}/qr.png` và `GET /trace/{trace_code}` tiếp tục pass, chứng minh form không làm thay đổi API tích hợp.

Nghiệm thu thủ công:

1. Khởi động service với `.env` có `PUBLIC_BASE_URL` là IPv4 LAN của máy và điện thoại cùng Wi-Fi.
2. Trên PC mở `/admin`, nhập một sản phẩm mới và API key, rồi chọn tạo QR.
3. Dùng điện thoại quét QR đang hiển thị; điện thoại phải mở trang `/trace/{trace_code}` với đúng sáu trường truy xuất.
4. Lặp lại với API key sai và `trace_code` vừa tạo để xác nhận hai lỗi `401` và `409` không tạo thêm dữ liệu.

Tính năng được coi là hoàn thành khi toàn bộ test tự động pass và một lần quét điện thoại mới mở đúng trang truy xuất của sản phẩm vừa nhập.
