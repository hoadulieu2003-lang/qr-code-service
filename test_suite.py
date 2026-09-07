#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Bond QR Code Generator - Automated Verification & Handoff Test Suite
Chạy kiểm thử tự động toàn diện các kịch bản biên, an ninh và hiệu năng.
Không yêu cầu thư viện ngoài (sử dụng chuẩn Python 3: urllib, concurrent.futures).
"""

import sys
import os
import time
import urllib.parse
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed

BASE_URL = os.environ.get("BOND_URL", "http://localhost:80")
SECRET = os.environ.get("BOND_SECRET", "a-very-long-and-complicated-secret")
MAX_SIZE = int(os.environ.get("BOND_MAX_SIZE", "1024"))

class TestResult:
    def __init__(self, name, category, passed, details, latency_ms=0):
        self.name = name
        self.category = category
        self.passed = passed
        self.details = details
        self.latency_ms = latency_ms

if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8')
if hasattr(sys.stderr, 'reconfigure'):
    sys.stderr.reconfigure(encoding='utf-8')

results = []

def record(name, category, passed, details, latency_ms=0):
    status = "[PASS]" if passed else "[FAIL]"
    print(f"{status} [{category}] {name} ({latency_ms:.1f}ms) - {details}")
    results.append(TestResult(name, category, passed, details, latency_ms))

def make_request(url, headers=None, method="GET"):
    req = urllib.request.Request(url, headers=headers or {}, method=method)
    start = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = resp.read()
            latency = (time.perf_counter() - start) * 1000
            return resp.status, resp.headers, data, latency
    except urllib.error.HTTPError as e:
        latency = (time.perf_counter() - start) * 1000
        return e.code, e.headers, e.read(), latency
    except Exception as e:
        latency = (time.perf_counter() - start) * 1000
        return 0, {}, str(e).encode(), latency

# ---------------------------------------------------------
# Test Cases
# ---------------------------------------------------------

def test_healthcheck():
    url = f"{BASE_URL}/health"
    status, headers, data, lat = make_request(url)
    passed = (status == 200)
    record("Healthcheck Endpoint", "Operational", passed, f"Status: {status}", lat)

def test_happy_path():
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "256",
        "content": "https://example.com/production"
    })
    url = f"{BASE_URL}/?{params}"
    status, headers, data, lat = make_request(url)
    
    is_png = data.startswith(b"\x89PNG\r\n\x1a\n")
    content_type = headers.get("Content-Type", "")
    passed = (status == 200 and is_png and "image/png" in content_type)
    record("Happy Path Basic QR", "Functional", passed, 
           f"Status: {status}, PNG Header: {is_png}, Content-Type: {content_type}, Bytes: {len(data)}", lat)

def test_secret_in_header_x_bond_secret():
    params = urllib.parse.urlencode({
        "size": "200",
        "content": "SecretInHeader"
    })
    url = f"{BASE_URL}/?{params}"
    headers = {"X-Bond-Secret": SECRET}
    status, _, data, lat = make_request(url, headers=headers)
    is_png = data.startswith(b"\x89PNG\r\n\x1a\n")
    passed = (status == 200 and is_png)
    record("Auth via X-Bond-Secret Header", "Security", passed, f"Status: {status}, Valid PNG: {is_png}", lat)

def test_secret_in_bearer_auth():
    params = urllib.parse.urlencode({
        "size": "200",
        "content": "SecretInBearer"
    })
    url = f"{BASE_URL}/?{params}"
    headers = {"Authorization": f"Bearer {SECRET}"}
    status, _, data, lat = make_request(url, headers=headers)
    is_png = data.startswith(b"\x89PNG\r\n\x1a\n")
    passed = (status == 200 and is_png)
    record("Auth via Authorization Bearer", "Security", passed, f"Status: {status}, Valid PNG: {is_png}", lat)

def test_invalid_secret():
    params = urllib.parse.urlencode({
        "secret": "wrong-secret-token",
        "size": "200",
        "content": "AttackPayload"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 403)
    record("Reject Invalid Secret", "Security", passed, f"Expected 403, Got: {status}", lat)

def test_missing_secret():
    params = urllib.parse.urlencode({
        "size": "200",
        "content": "NoSecretPayload"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 403)
    record("Reject Missing Secret", "Security", passed, f"Expected 403, Got: {status}", lat)

def test_invalid_size_string():
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "not-a-number",
        "content": "ValidContent"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, data, lat = make_request(url)
    # Must NOT be 500 Internal Server Error
    passed = (status == 400)
    record("Input Validation: String as Size", "Boundary", passed, f"Expected 400 (Bad Request), Got: {status} - Body: {data.decode(errors='ignore')[:60]}", lat)

def test_negative_size():
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "-200",
        "content": "ValidContent"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 400)
    record("Input Validation: Negative Size", "Boundary", passed, f"Expected 400, Got: {status}", lat)

def test_zero_size():
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "0",
        "content": "ValidContent"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 400)
    record("Input Validation: Zero Size", "Boundary", passed, f"Expected 400, Got: {status}", lat)

def test_size_exceeding_max():
    oversized = MAX_SIZE + 500
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": str(oversized),
        "content": "ValidContent"
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 400)
    record("Input Validation: Size > MAX_SIZE", "Boundary", passed, f"Expected 400, Got: {status}", lat)

def test_missing_content():
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "256",
        "content": ""
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    passed = (status == 400)
    record("Input Validation: Empty Content", "Boundary", passed, f"Expected 400, Got: {status}", lat)

def test_nested_url_query():
    nested_url = "https://example.com/checkout?orderId=98765&discountCode=SUMMER2026&ref=affiliate_88"
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "300",
        "content": nested_url
    })
    url = f"{BASE_URL}/?{params}"
    status, _, data, lat = make_request(url)
    is_png = data.startswith(b"\x89PNG\r\n\x1a\n")
    passed = (status == 200 and is_png)
    record("Nested Query String Content", "Encoding", passed, f"Status: {status}, Valid PNG: {is_png}", lat)

def test_unicode_vietnamese_and_emoji():
    vietnamese_text = "Hệ thống xác thực mã QR: Đã bàn giao nghiệm thu thành công! 🇻🇳 🚀 ⭐ [Tiếng Việt có dấu]"
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "350",
        "content": vietnamese_text
    })
    url = f"{BASE_URL}/?{params}"
    status, _, data, lat = make_request(url)
    is_png = data.startswith(b"\x89PNG\r\n\x1a\n")
    passed = (status == 200 and is_png)
    record("UTF-8 Diacritics & Emoji Content", "Encoding", passed, f"Status: {status}, Valid PNG: {is_png}", lat)

def test_qr_capacity_overflow():
    # Chuỗi 10,000 ký tự vượt ngưỡng tối đa của QR Code Version 40 (tối đa ~7089 ký tự)
    huge_payload = "A" * 10000
    params = urllib.parse.urlencode({
        "secret": SECRET,
        "size": "400",
        "content": huge_payload
    })
    url = f"{BASE_URL}/?{params}"
    status, _, _, lat = make_request(url)
    # Phải trả về 400 Bad Request (qrcode.Encode trả lỗi), không được crash server 500
    passed = (status == 400)
    record("Payload Capacity Overflow (>7K chars)", "Boundary", passed, f"Expected 400, Got: {status}", lat)

def test_concurrency_stress(concurrency=50, total_requests=100):
    print(f"\n--- Bắt đầu kiểm thử Chịu tải đồng thời: {total_requests} requests qua {concurrency} workers ---")
    
    url = f"{BASE_URL}/?{urllib.parse.urlencode({'secret': SECRET, 'size': '256', 'content': 'StressTestPayloadBench'})}"
    latencies = []
    successes = 0
    failures = 0
    start_time = time.perf_counter()
    
    def worker():
        s, h, d, lat = make_request(url)
        return (s == 200 and d.startswith(b"\x89PNG\r\n\x1a\n")), lat

    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = [executor.submit(worker) for _ in range(total_requests)]
        for f in as_completed(futures):
            ok, lat = f.result()
            latencies.append(lat)
            if ok:
                successes += 1
            else:
                failures += 1
                
    total_duration = time.perf_counter() - start_time
    rps = total_requests / total_duration if total_duration > 0 else 0
    avg_lat = sum(latencies) / len(latencies) if latencies else 0
    min_lat = min(latencies) if latencies else 0
    max_lat = max(latencies) if latencies else 0
    
    passed = (successes == total_requests and failures == 0)
    details = f"Success: {successes}/{total_requests} (100%), RPS: {rps:.1f} req/s, Min: {min_lat:.1f}ms, Avg: {avg_lat:.1f}ms, Max: {max_lat:.1f}ms"
    record(f"Concurrency Stress ({concurrency} workers, {total_requests} reqs)", "Stress", passed, details, avg_lat)

def generate_markdown_report(output_path="handoff_report.md"):
    total = len(results)
    passed = sum(1 for r in results if r.passed)
    failed = total - passed
    pass_rate = (passed / total * 100) if total > 0 else 0

    lines = [
        "# Báo Cáo Nghiệm Thu Kỹ Thuật Module QR Code (Bond Handoff Report)",
        "",
        f"- **Thời gian chạy nghiệm thu**: {time.strftime('%Y-%m-%d %H:%M:%S')}",
        f"- **Mục tiêu kiểm thử**: `{BASE_URL}`",
        f"- **Tổng số ca kiểm thử**: **{total}**",
        f"- **Đạt chuẩn (Passed)**: **{passed}**",
        f"- **Thất bại (Failed)**: **{failed}**",
        f"- **Tỷ lệ vượt qua (Pass Rate)**: **{pass_rate:.1f}%**",
        "",
        "## Bảng Chi Tiết Kết Quả Kiểm Thử Từng Kịch Bản",
        "",
        "| Danh mục | Kịch bản kiểm thử | Trạng thái | Độ trễ (ms) | Chi tiết kết quả |",
        "| :--- | :--- | :---: | :---: | :--- |"
    ]

    for r in results:
        badge = "✅ PASS" if r.passed else "❌ FAIL"
        lines.append(f"| {r.category} | {r.name} | {badge} | {r.latency_ms:.1f} | {r.details} |")

    lines.extend([
        "",
        "## Kết luận Nghiệm thu Bàn giao (Handoff Verdict)",
        ""
    ])

    if failed == 0:
        lines.append("> **KẾT LUẬN: ĐẠT CHUẨN BÀN GIAO (READY FOR PRODUCTION HANDOFF)**\n>\n> Toàn bộ các tiêu chí về an ninh mạng, xác thực dữ liệu đầu vào, chuẩn hóa mã lỗi HTTP, tương thích UTF-8 và khả năng chịu tải đồng thời đều đã vượt qua 100%.")
    else:
        lines.append(f"> ⚠️ **KẾT LUẬN: CHƯA ĐẠT CHUẨN** — Có {failed} ca kiểm thử thất bại cần được khắc phục trước khi bàn giao.")

    report_content = "\n".join(lines)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(report_content)
    print(f"\n[Báo Cáo] Đã tạo thành công file báo cáo nghiệm thu: {output_path}")

def main():
    print("==================================================================")
    print("  BOND QR CODE GENERATOR - AUTOMATED VERIFICATION TEST SUITE     ")
    print(f"  Target: {BASE_URL} | Secret: {'*' * len(SECRET)}")
    print("==================================================================\n")

    # Kiểm tra kết nối sơ bộ
    try:
        urllib.request.urlopen(f"{BASE_URL}/health", timeout=3)
    except Exception as e:
        print(f"❌ Không thể kết nối tới {BASE_URL}. Vui lòng đảm bảo server Bond đang chạy!")
        print(f"   Chi tiết lỗi: {e}")
        sys.exit(1)

    # 1. Operational & Functional
    test_healthcheck()
    test_happy_path()

    # 2. Security
    test_secret_in_header_x_bond_secret()
    test_secret_in_bearer_auth()
    test_invalid_secret()
    test_missing_secret()

    # 3. Boundary & Input Validation
    test_invalid_size_string()
    test_negative_size()
    test_zero_size()
    test_size_exceeding_max()
    test_missing_content()
    test_qr_capacity_overflow()

    # 4. Encoding & Special Chars
    test_nested_url_query()
    test_unicode_vietnamese_and_emoji()

    # 5. Stress Testing
    test_concurrency_stress(concurrency=50, total_requests=100)

    # 6. Export Report
    generate_markdown_report("handoff_report.md")

if __name__ == "__main__":
    main()
