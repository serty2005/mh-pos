import pathlib
import sys
import unittest
import io
from contextlib import redirect_stdout
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "lib"))


class FakeClient:
    def __init__(self, name):
        self.name = name
        self.posts = []
        self.gets = []
        self.registered_pairing = {}

    def root_get(self, path, headers=None, expected_status=(200,)):
        self.gets.append((path, headers or {}, tuple(expected_status)))
        return {"status": "ok", "service": self.name}

    def get(self, path, headers=None, expected_status=(200,)):
        self.gets.append((path, headers or {}, tuple(expected_status)))
        if path == "/system/provisioning-status":
            return {"node_device_id": "node-1", "paired": False, "status": "not_configured"}
        raise AssertionError(f"unexpected GET {path}")

    def post(self, path, body=None, headers=None, expected_status=(200, 201)):
        self.posts.append((path, body or {}, headers or {}, tuple(expected_status)))
        if path == "/pairing-codes":
            self.registered_pairing = dict(body)
            return {"status": "registered", "expires_at": body["expires_at"]}
        if path == "/pairing-codes/resolve":
            if len([item for item in self.posts if item[0] == "/pairing-codes/resolve"]) == 1:
                return {
                    "cloud_url": self.registered_pairing["cloud_url"],
                    "restaurant_id": self.registered_pairing["restaurant_id"],
                    "node_device_id": body["node_device_id"],
                    "credentials": {"type": "node_token", "token": "secret-token"},
                }
            raise RuntimeError("HTTP 409 PAIRING_CODE_INVALID")
        raise AssertionError(f"unexpected POST {path}")


class StackSmokeTest(unittest.TestCase):
    def test_parse_suites_expands_all_and_comma_separated_values(self):
        from mhpos_stack import parse_suites

        self.assertEqual(
            parse_suites(["health,license_pairing", "cloud_to_edge_masterdata"]),
            ["health", "license_pairing", "cloud_to_edge_masterdata"],
        )
        self.assertEqual(parse_suites(["all"]), ["health", "license_pairing", "cloud_to_edge_masterdata"])

    def test_health_suite_checks_cloud_pos_and_license(self):
        from mhpos_stack import StackContext, run_health_suite

        ctx = StackContext(FakeClient("cloud"), FakeClient("pos"), FakeClient("license"))
        result = run_health_suite(ctx)

        self.assertEqual(result["name"], "health")
        self.assertEqual(result["status"], "passed")
        self.assertEqual(result["details"]["cloud"]["response"]["status"], "ok")
        self.assertEqual(result["details"]["pos"]["response"]["status"], "ok")
        self.assertEqual(result["details"]["license"]["response"]["status"], "ok")

    def test_health_suite_reports_each_service_when_one_fails(self):
        from mhpos_stack import StackContext, run_health_suite

        class BrokenClient(FakeClient):
            def root_get(self, path, headers=None, expected_status=(200,)):
                raise RuntimeError("cloud unavailable")

        ctx = StackContext(BrokenClient("cloud"), FakeClient("pos"), FakeClient("license"))
        result = run_health_suite(ctx)

        self.assertEqual(result["status"], "failed")
        self.assertEqual(result["details"]["cloud"]["status"], "failed")
        self.assertEqual(result["details"]["pos"]["status"], "passed")
        self.assertEqual(result["details"]["license"]["status"], "passed")

    def test_license_pairing_suite_registers_resolves_and_checks_consumed_code(self):
        from mhpos_stack import StackContext, run_license_pairing_suite

        license_client = FakeClient("license")
        ctx = StackContext(FakeClient("cloud"), FakeClient("pos"), license_client)
        result = run_license_pairing_suite(ctx)

        self.assertEqual(result["name"], "license_pairing")
        self.assertEqual(result["status"], "passed")
        self.assertEqual([path for path, *_ in license_client.posts], ["/pairing-codes", "/pairing-codes/resolve", "/pairing-codes/resolve"])
        self.assertEqual(result["details"]["second_resolve_rejected"], True)
        self.assertNotIn("secret-token", str(result))

    def test_run_selected_suites_converts_exception_to_failed_result(self):
        from mhpos_stack import StackContext, run_selected_suites

        class BrokenClient(FakeClient):
            def root_get(self, path, headers=None, expected_status=(200,)):
                raise RuntimeError("boom")

        ctx = StackContext(BrokenClient("cloud"), FakeClient("pos"), FakeClient("license"))
        result = run_selected_suites(ctx, ["health"])

        self.assertEqual(result["status"], "failed")
        self.assertEqual(result["suites"][0]["status"], "failed")
        self.assertIn("health failed for cloud", result["suites"][0]["error"])
        self.assertIn("boom", result["suites"][0]["details"]["cloud"]["error"])

    def test_masterdata_suite_fails_before_seed_when_edge_is_already_paired(self):
        from mhpos_stack import StackContext, run_selected_suites

        class PairedPOS(FakeClient):
            def get(self, path, headers=None, expected_status=(200,)):
                if path == "/system/provisioning-status":
                    return {"node_device_id": "node-1", "paired": True, "restaurant_id": "old-restaurant"}
                return super().get(path, headers=headers, expected_status=expected_status)

        cloud = FakeClient("cloud")
        ctx = StackContext(cloud, PairedPOS("pos"), FakeClient("license"))
        result = run_selected_suites(ctx, ["cloud_to_edge_masterdata"])

        self.assertEqual(result["status"], "failed")
        self.assertIn("already paired", result["suites"][0]["error"])
        self.assertEqual(cloud.posts, [])

    def test_masterdata_suite_reuses_matching_summary_when_edge_is_already_paired(self):
        import mhpos_stack
        from mhpos_stack import StackContext, run_selected_suites

        class PairedPOS(FakeClient):
            def get(self, path, headers=None, expected_status=(200,)):
                if path == "/system/provisioning-status":
                    return {"node_device_id": "node-1", "paired": True, "restaurant_id": "restaurant-1"}
                return super().get(path, headers=headers, expected_status=expected_status)

        summary = {
            "restaurant_id": "restaurant-1",
            "node_device_id": "node-1",
            "manager_pin": "2222",
            "manager_employee_id": "manager-1",
        }
        cloud = FakeClient("cloud")
        ctx = StackContext(cloud, PairedPOS("pos"), FakeClient("license"))

        with mock.patch.object(
            mhpos_stack,
            "verify_pos_read_model",
            return_value={"manager_employee_id": "manager-1", "headers": {"X-Session-ID": "session-1"}},
        ), mock.patch.object(
            mhpos_stack,
            "create_post_pairing_sync_item",
            return_value={"menu_item_id": "menu-1", "publication_id": "publication-1"},
        ) as create_sync_item, mock.patch.object(
            mhpos_stack,
            "wait_for_menu_item",
            return_value={"id": "menu-1"},
        ), mock.patch.object(
            mhpos_stack,
            "verify_sync_status",
            return_value={"sync_status": {"status": "ok"}},
        ):
            result = run_selected_suites(ctx, ["cloud_to_edge_masterdata"], existing_summary=summary)

        self.assertEqual(result["status"], "passed")
        self.assertEqual(result["suites"][0]["details"]["reused_existing_pairing"], True)
        create_sync_item.assert_called_once()
        self.assertEqual(cloud.posts, [])

    def test_masterdata_suite_reports_sync_status_when_post_pairing_item_times_out(self):
        import mhpos_stack
        from mhpos_stack import StackContext, run_selected_suites

        class PairedPOS(FakeClient):
            def get(self, path, headers=None, expected_status=(200,)):
                if path == "/system/provisioning-status":
                    return {"node_device_id": "node-1", "paired": True, "restaurant_id": "restaurant-1"}
                return super().get(path, headers=headers, expected_status=expected_status)

        summary = {
            "restaurant_id": "restaurant-1",
            "node_device_id": "node-1",
            "manager_pin": "2222",
            "manager_employee_id": "manager-1",
        }
        ctx = StackContext(FakeClient("cloud"), PairedPOS("pos"), FakeClient("license"))

        with mock.patch.object(
            mhpos_stack,
            "verify_pos_read_model",
            return_value={"manager_employee_id": "manager-1", "headers": {"X-Session-ID": "session-1"}},
        ), mock.patch.object(
            mhpos_stack,
            "create_post_pairing_sync_item",
            return_value={"menu_item_id": "menu-1", "publication_id": "publication-1"},
        ), mock.patch.object(
            mhpos_stack,
            "wait_for_menu_item",
            side_effect=TimeoutError("condition was not met before timeout"),
        ), mock.patch.object(
            mhpos_stack,
            "verify_sync_status",
            return_value={
                "sync_status": {"pending": 2, "failed": 0},
                "outbox": [{"last_error": "cloud exchange returned HTTP 403: SYNC_FORBIDDEN"}],
            },
        ):
            result = run_selected_suites(ctx, ["cloud_to_edge_masterdata"], existing_summary=summary)

        self.assertEqual(result["status"], "failed")
        self.assertIn("post-pairing Cloud->Edge menu item did not sync", result["suites"][0]["error"])
        self.assertIn("SYNC_FORBIDDEN", result["suites"][0]["error"])

    def test_stack_cli_returns_non_zero_when_suite_failed(self):
        script_path = ROOT / "run-stack-smoke.py"
        namespace = {"__file__": str(script_path)}
        exec(compile(script_path.read_text(encoding="utf-8"), str(script_path), "exec"), namespace)

        failed = {"status": "failed", "suites": [{"name": "health", "status": "failed", "details": {}, "error": "boom"}]}

        with mock.patch.dict(namespace, {"run_selected_suites": lambda *args, **kwargs: failed}):
            with redirect_stdout(io.StringIO()):
                exit_code = namespace["main"](["--suite", "health"])

        self.assertEqual(exit_code, 1)


if __name__ == "__main__":
    unittest.main()
