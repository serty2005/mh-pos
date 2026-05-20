import pathlib
import sys
import unittest
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "lib"))


class HttpHelpersTest(unittest.TestCase):
    def test_normalize_base_url_strips_api_v1_and_trailing_slash(self):
        from mhpos_http import normalize_base_url

        self.assertEqual(
            normalize_base_url(" http://localhost:8090/api/v1/ "),
            "http://localhost:8090",
        )

    def test_api_url_adds_api_prefix_once(self):
        from mhpos_http import api_url

        self.assertEqual(
            api_url("http://localhost:8090/api/v1", "/restaurants"),
            "http://localhost:8090/api/v1/restaurants",
        )

    def test_request_wraps_socket_timeout_as_http_error(self):
        from mhpos_http import HttpError, JsonClient

        client = JsonClient("http://example.test:8090", timeout_seconds=1)

        with mock.patch("urllib.request.urlopen", side_effect=TimeoutError("timed out")):
            with self.assertRaises(HttpError) as raised:
                client.get("/restaurants")

        self.assertEqual(raised.exception.status, 0)
        self.assertIn("timed out", raised.exception.body)

    def test_request_bypasses_environment_proxy_for_loopback_url(self):
        from mhpos_http import JsonClient

        class FakeResponse:
            def __enter__(self):
                return self

            def __exit__(self, exc_type, exc, tb):
                return False

            def read(self):
                return b'{"status":"ok"}'

            def getcode(self):
                return 200

        fake_opener = mock.Mock()
        fake_opener.open.return_value = FakeResponse()
        client = JsonClient("http://127.0.0.1:8090", timeout_seconds=1)

        with mock.patch("urllib.request.urlopen", side_effect=AssertionError("proxy-aware urlopen must not be used")):
            with mock.patch("urllib.request.build_opener", return_value=fake_opener) as build_opener:
                result = client.root_get("/health")

        self.assertEqual(result, {"status": "ok"})
        proxy_handler = build_opener.call_args.args[0]
        self.assertEqual(proxy_handler.proxies, {})


if __name__ == "__main__":
    unittest.main()
