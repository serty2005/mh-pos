import json
import time
import urllib.error
import urllib.parse
import urllib.request


class HttpError(RuntimeError):
    def __init__(self, method, url, status, body):
        super().__init__(f"{method} {url} failed with HTTP {status}: {body}")
        self.method = method
        self.url = url
        self.status = status
        self.body = body


def normalize_base_url(value):
    url = str(value or "").strip().rstrip("/")
    if url.endswith("/api/v1"):
        url = url[: -len("/api/v1")]
    return url.rstrip("/")


def api_url(base_url, path):
    base = normalize_base_url(base_url)
    clean_path = "/" + str(path or "").lstrip("/")
    return base + "/api/v1" + clean_path


def root_url(base_url, path):
    base = normalize_base_url(base_url)
    clean_path = "/" + str(path or "").lstrip("/")
    return base + clean_path


class JsonClient:
    def __init__(self, base_url, timeout_seconds=20):
        self.base_url = normalize_base_url(base_url)
        self.timeout_seconds = timeout_seconds

    def get(self, path, headers=None, expected_status=(200,)):
        return self.request("GET", path, None, headers, expected_status)

    def root_get(self, path, headers=None, expected_status=(200,)):
        return self.request("GET", path, None, headers, expected_status, api_prefix=False)

    def post(self, path, body=None, headers=None, expected_status=(200, 201)):
        return self.request("POST", path, body or {}, headers, expected_status)

    def put(self, path, body=None, headers=None, expected_status=(200, 201)):
        return self.request("PUT", path, body or {}, headers, expected_status)

    def patch(self, path, body=None, headers=None, expected_status=(200,)):
        return self.request("PATCH", path, body or {}, headers, expected_status)

    def request(self, method, path, body=None, headers=None, expected_status=(200,), api_prefix=True):
        url = api_url(self.base_url, path) if api_prefix else root_url(self.base_url, path)
        data = None
        request_headers = {"Accept": "application/json"}
        if headers:
            request_headers.update(headers)
        if body is not None:
            data = json.dumps(body, ensure_ascii=False).encode("utf-8")
            request_headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data=data, headers=request_headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=self.timeout_seconds) as resp:
                raw = resp.read()
                status = resp.getcode()
        except urllib.error.HTTPError as exc:
            raw = exc.read()
            status = exc.code
            text = raw.decode("utf-8", errors="replace")
            raise HttpError(method, url, status, text) from exc
        except urllib.error.URLError as exc:
            raise HttpError(method, url, 0, str(exc.reason)) from exc
        if status not in tuple(expected_status):
            text = raw.decode("utf-8", errors="replace")
            raise HttpError(method, url, status, text)
        if not raw:
            return None
        text = raw.decode("utf-8")
        return json.loads(text)


def wait_until(callback, timeout_seconds=60, interval_seconds=2):
    deadline = time.monotonic() + timeout_seconds
    last_error = None
    while True:
        try:
            value = callback()
            if value:
                return value
        except Exception as exc:
            last_error = exc
        if time.monotonic() >= deadline:
            if last_error:
                raise TimeoutError(f"condition was not met before timeout: {last_error}") from last_error
            raise TimeoutError("condition was not met before timeout")
        if interval_seconds > 0:
            time.sleep(interval_seconds)
