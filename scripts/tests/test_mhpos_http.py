import pathlib
import sys
import unittest


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


if __name__ == "__main__":
    unittest.main()
