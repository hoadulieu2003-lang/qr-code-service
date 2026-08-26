# Hướng dẫn kiểm thử bàn giao QR truy xuất nguồn gốc

Mục đích: xác nhận chuỗi hoàn chỉnh **nhập sản phẩm → sinh QR → điện thoại quét → xem đúng thông tin công khai**, đồng thời kiểm tra API có thể được hệ thống khác gọi an toàn.

## Chuẩn bị

1. Hoàn tất [khởi động nhanh](01-quick-start.md), giữ server chạy ở PowerShell foreground.
2. Chạy `.\scripts\check-health.ps1 -Port 18080` và nhận dòng `PASS`.
3. Bật Wi-Fi cùng mạng trên máy tính và điện thoại; tắt mobile data tạm thời nếu điện thoại tự chuyển mạng.
4. Mở `http://localhost:18080/admin` **trên máy chạy server**. Việc mở `/admin` bằng IP LAN phải bị chặn `403`.

## A. Kiểm thử bằng form, không cần API key

Nhập một sản phẩm mới, ví dụ:

| Trường | Giá trị thử |
| --- | --- |
| Mã truy xuất | `SP-FORM-KIEM-THU-001` |
| Mã sản phẩm | `CP-250G` |
| Tên sản phẩm | `Cà phê rang xay kiểm thử` |
| Mã lô | `LO-FORM-001` |
| Ngày sản xuất | `2026-08-26` |
| Hạn dùng | `2027-08-26` |
| Xuất xứ | `Đắk Lắk, Việt Nam` |
| Trạng thái | `verified` |

1. Nhấn **Tạo QR**. Kỳ vọng: kết quả hiển thị QR và URL `/trace/SP-FORM-KIEM-THU-001` dùng địa chỉ IPv4 LAN, không phải `localhost`.
2. Không nhập API key ở bất cứ đâu trong form. Kỳ vọng: form vẫn tạo thành công.
3. Đổi hạn dùng sớm hơn ngày sản xuất, thử lại với mã mới. Kỳ vọng: lỗi validation, không có QR/sản phẩm mới.
4. Gửi lại đúng cùng `trace_code`. Kỳ vọng: `409 Conflict`; sản phẩm đầu tiên không bị ghi đè.

## B. Kiểm thử QR bằng điện thoại

1. Dùng camera hoặc ứng dụng quét QR trên điện thoại để quét **ảnh QR hiển thị ở form**.
2. Kỳ vọng: điện thoại mở trang `http://<IPv4-may-chay>:18080/trace/SP-FORM-KIEM-THU-001`.
3. Trang công khai phải hiện đúng 6 nhóm thông tin: tên sản phẩm; mã sản phẩm và mã lô; ngày sản xuất; hạn dùng; xuất xứ; trạng thái xác thực.
4. So sánh toàn bộ giá trị với form, bao gồm tiếng Việt có dấu.

Nếu quét không mở được trang, kiểm tra lần lượt: `PUBLIC_BASE_URL` có đúng IPv4 Wi-Fi không; hai thiết bị có cùng Wi-Fi không; Windows Firewall có cho phép mạng Private không; server còn chạy/`/health` có `200` không. Kiểm tra server hoặc PNG bằng máy tính **không thay thế** việc quét thật bằng điện thoại.

## C. Kiểm thử API tích hợp

Tạo mã mới mỗi lần để không vướng `409`:

```powershell
.\scripts\test-api.ps1 -TraceCode ("SP-API-" + (Get-Date -Format 'yyyyMMddHHmmss'))
```

Kỳ vọng: output chỉ có `trace_code`, `trace_url`, `qr_url`; `trace_url` trả `200`; API key không xuất hiện trong output. Sau đó theo [API integration](02-api-integration.md) để kiểm tra thêm: API không key trả `401`, request hợp lệ trả `201`, mã trùng trả `409`, và tải QR PNG bằng header `X-API-Key`.

## D. Mười kịch bản nghiệm thu bắt buộc

Thực hiện qua API với mỗi `trace_code` duy nhất hoặc qua form khi phù hợp. Sau mỗi case hợp lệ, tải QR và quét ít nhất một QR bằng điện thoại; các case còn lại có thể kiểm tra URL trace/HTTP để tiết kiệm thời gian.

| # | Kịch bản | Dữ liệu/cách làm | Kỳ vọng |
| --- | --- | --- | --- |
| 1 | Sản phẩm thông thường | Tám trường hợp lệ, `verified` | `201`, QR PNG 512×512, trace `200` |
| 2 | Tiếng Việt/Unicode | Tên và xuất xứ có dấu | `201`; trang trace giữ đúng ký tự |
| 3 | Ký tự HTML | Tên chứa `<`, `>`, `&`, `"` | `201`; trang hiển thị text an toàn, không chạy HTML |
| 4 | Chưa xác thực | `verification_status: "unverified"` | `201`; trang thể hiện trạng thái `unverified` |
| 5 | Khoảng trắng dư | Bao quanh `trace_code` bằng khoảng trắng | `201`; mã trong response được trim |
| 6 | Trace code dài hợp lệ | Chuỗi chữ/số/`-`/`_` dài nhưng QR còn sức chứa | `201`; QR và trace hoạt động |
| 7 | Ngày bằng nhau | `manufactured_at` bằng `expires_at` | `201`; ngày bằng nhau được chấp nhận |
| 8 | Sai thứ tự ngày | Hạn dùng trước ngày sản xuất | `400`; không tạo QR, `/trace/{traceCode}` là `404` |
| 9 | Mã trùng | Gửi cùng `trace_code` hai lần | Lần 1 `201`, lần 2 `409`; dữ liệu cũ còn nguyên |
| 10 | QR vượt sức chứa | `PUBLIC_BASE_URL`/mã cực dài làm QR không mã hóa được | `500`; không lưu sản phẩm, trace là `404` |

## E. Ghi biên bản nghiệm thu

Ghi ngày test, máy chủ/IP/port, mã test, người quét điện thoại và kết quả từng case. Không ghi `API_KEY` vào biên bản. Mẫu checklist chính thức nằm ở [04-handover-checklist.md](04-handover-checklist.md).
