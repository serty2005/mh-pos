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
        self.assertEqual(
            parse_suites(["all"]),
            [
                "health",
                "license_pairing",
                "cloud_to_edge_masterdata",
                "pos_cashier_runtime",
                "pos_refund_after_shift_close",
            ],
        )

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

    def test_pos_cashier_runtime_suite_runs_full_backend_path_and_hides_sessions(self):
        from mhpos_http import HttpError
        from mhpos_stack import StackContext, run_pos_cashier_runtime_suite

        class RuntimePOS(FakeClient):
            def __init__(self):
                super().__init__("pos")
                self.patches = []

            def post(self, path, body=None, headers=None, expected_status=(200, 201)):
                self.posts.append((path, body or {}, headers or {}, tuple(expected_status)))
                if path == "/auth/pin-login":
                    return {"session": {"id": "session-manager"}, "actor": {"employee_id": "manager-1"}}
                if path == "/employee-shifts/open":
                    return {"id": "shift-1", "opened_by_employee_id": body["opened_by_employee_id"]}
                if path == "/cash-shifts/open":
                    return {"id": "cash-1", "shift_id": "shift-1", "opened_by_employee_id": body["opened_by_employee_id"]}
                if path == "/orders":
                    return {"id": "order-1", "status": "open"}
                if path == "/orders/order-1/lines":
                    if body["menu_item_id"] == "menu-mod":
                        return {"id": "line-mod", "menu_item_id": body["menu_item_id"]}
                    if body["menu_item_id"] == "menu-service":
                        return {"id": "line-service", "menu_item_id": body["menu_item_id"]}
                    return {"id": "line-normal", "menu_item_id": body["menu_item_id"]}
                if path == "/orders/order-1/precheck":
                    return {"id": "precheck-1", "total": 35000, "remaining_total": 35000, "currency_code": "RUB"}
                if path == "/prechecks/precheck-1/payments":
                    return {"id": "payment-1", "status": "captured"}
                if path == "/checks/check-1/reprint":
                    return {"id": "reprint-1", "document_type": "check"}
                if path == "/checks/check-1/cancellations":
                    return {
                        "id": "operation-1",
                        "operation_type": "cancellation",
                        "operation_kind": body["operation_kind"],
                    }
                return super().post(path, body=body, headers=headers, expected_status=expected_status)

            def patch(self, path, body=None, headers=None, expected_status=(200,)):
                self.patches.append((path, body or {}, headers or {}, tuple(expected_status)))
                if path == "/orders/order-1/lines/line-mod/modifiers":
                    return {"id": "line-mod", "modifiers": body["selected_modifiers"]}
                raise AssertionError(f"unexpected PATCH {path}")

            def get(self, path, headers=None, expected_status=(200,)):
                self.gets.append((path, headers or {}, tuple(expected_status)))
                if path == "/system/provisioning-status":
                    return {"node_device_id": "node-1", "paired": True, "restaurant_id": "restaurant-1"}
                if path == "/employee-shifts/current":
                    return None
                if path == "/cash-shifts/current":
                    raise HttpError("GET", path, 404, "{}")
                if path.startswith("/halls"):
                    return [{"id": "hall-1", "name": "Main"}]
                if path.startswith("/tables"):
                    return [{"id": "table-1", "name": "T1"}]
                if path == "/menu/items":
                    return [
                        {"id": "menu-normal", "item_type": "dish", "active": True},
                        {
                            "id": "menu-mod",
                            "item_type": "dish",
                            "active": True,
                            "modifier_groups": [
                                {
                                    "id": "modifier-group-1",
                                    "active": True,
                                    "max_count": 1,
                                    "options": [{"id": "modifier-option-1", "active": True}],
                                }
                            ],
                        },
                        {"id": "menu-service", "item_type": "service", "active": True},
                    ]
                if path == "/orders/order-1":
                    return {"id": "order-1", "status": "closed", "check": {"id": "check-1", "status": "paid"}}
                if path.startswith("/orders/closed"):
                    return [{"id": "order-1", "status": "closed", "check": {"id": "check-1"}}]
                if path == "/checks/check-1":
                    return {"id": "check-1", "status": "paid", "total": 35000}
                if path == "/checks/check-1/financial-operations":
                    return [{"id": "operation-1", "operation_type": "cancellation"}]
                if path == "/storage/status":
                    return {
                        "retention_mode": "dry_run_only",
                        "database_size_bytes": 4096,
                        "counts": {"closed_orders": 1, "checks": 1, "financial_operations": 1},
                    }
                raise AssertionError(f"unexpected GET {path}")

        summary = {
            "restaurant_id": "restaurant-1",
            "node_device_id": "node-1",
            "manager_pin": "2222",
            "manager_employee_id": "manager-1",
            "cashier_pin": "1111",
            "cashier_employee_id": "cashier-1",
            "hall_id": "hall-1",
            "table_ids": ["table-1"],
            "menu_item_ids": ["menu-normal", "menu-mod", "menu-service"],
        }
        pos = RuntimePOS()
        ctx = StackContext(FakeClient("cloud"), pos, FakeClient("license"))

        result = run_pos_cashier_runtime_suite(ctx, existing_summary=summary, client_device_id="client-1")

        self.assertEqual(result["name"], "pos_cashier_runtime")
        self.assertEqual(result["status"], "passed")
        self.assertEqual(result["details"]["order_id"], "order-1")
        self.assertEqual(result["details"]["modifier_line_added"], True)
        self.assertEqual(result["details"]["service_line_added"], True)
        payment_posts = [body for path, body, *_ in pos.posts if path == "/prechecks/precheck-1/payments"]
        cancellation_posts = [body for path, body, *_ in pos.posts if path == "/checks/check-1/cancellations"]
        self.assertTrue(payment_posts[0]["command_id"].startswith("capture-payment-"))
        self.assertTrue(cancellation_posts[0]["command_id"].startswith("record-cancellation-"))
        self.assertNotIn("session-manager", str(result))

    def test_pos_refund_after_shift_close_suite_closes_original_shifts_and_records_refund(self):
        from mhpos_http import HttpError
        from mhpos_stack import StackContext, run_pos_refund_after_shift_close_suite

        class RefundPOS(FakeClient):
            def __init__(self):
                super().__init__("pos")
                self.manager_shift_open = False
                self.shift_sequence = 0
                self.current_cash_shift = None

            def post(self, path, body=None, headers=None, expected_status=(200, 201)):
                self.posts.append((path, body or {}, headers or {}, tuple(expected_status)))
                if path == "/auth/pin-login":
                    employee_id = "manager-1" if body["pin"] == "2222" else "cashier-1"
                    return {"session": {"id": "session-" + employee_id}, "actor": {"employee_id": employee_id}}
                if path == "/employee-shifts/open":
                    self.shift_sequence += 1
                    shift_id = "shift-sale" if self.shift_sequence == 1 else "shift-refund"
                    self.manager_shift_open = True
                    return {"id": shift_id, "status": "open", "opened_by_employee_id": body["opened_by_employee_id"]}
                if path == "/cash-shifts/open":
                    shift_id = "shift-sale" if self.shift_sequence == 1 else "shift-refund"
                    cash_id = "cash-sale" if self.shift_sequence == 1 else "cash-refund"
                    self.current_cash_shift = {
                        "id": cash_id,
                        "status": "open",
                        "shift_id": shift_id,
                        "opened_by_employee_id": body["opened_by_employee_id"],
                    }
                    return dict(self.current_cash_shift)
                if path == "/orders":
                    return {"id": "order-1", "status": "open"}
                if path == "/orders/order-1/lines":
                    return {"id": "line-normal", "menu_item_id": body["menu_item_id"]}
                if path == "/orders/order-1/precheck":
                    return {"id": "precheck-1", "total": 15000, "remaining_total": 15000, "currency_code": "RUB"}
                if path == "/prechecks/precheck-1/payments":
                    return {"id": "payment-1", "status": "captured"}
                if path == "/cash-shifts/cash-sale/close":
                    self.current_cash_shift = None
                    return {"id": "cash-sale", "status": "closed"}
                if path == "/employee-shifts/shift-sale/close":
                    self.manager_shift_open = False
                    return {"id": "shift-sale", "status": "closed"}
                if path == "/checks/check-1/refunds":
                    return {
                        "id": "operation-refund-1",
                        "operation_type": "refund",
                        "operation_kind": body["operation_kind"],
                        "amount": 15000,
                    }
                return super().post(path, body=body, headers=headers, expected_status=expected_status)

            def get(self, path, headers=None, expected_status=(200,)):
                self.gets.append((path, headers or {}, tuple(expected_status)))
                if path == "/system/provisioning-status":
                    return {"node_device_id": "node-1", "paired": True, "restaurant_id": "restaurant-1"}
                if path == "/employee-shifts/current":
                    if self.manager_shift_open:
                        shift_id = "shift-sale" if self.shift_sequence == 1 else "shift-refund"
                        return {"id": shift_id, "status": "open", "opened_by_employee_id": "manager-1"}
                    return None
                if path == "/cash-shifts/current":
                    if self.current_cash_shift:
                        return dict(self.current_cash_shift)
                    raise HttpError("GET", path, 404, "{}")
                if path.startswith("/halls"):
                    return [{"id": "hall-1", "name": "Main"}]
                if path.startswith("/tables"):
                    return [{"id": "table-1", "name": "T1"}]
                if path == "/menu/items":
                    return [{"id": "menu-normal", "item_type": "dish", "active": True}]
                if path == "/orders/order-1":
                    return {"id": "order-1", "status": "closed", "check": {"id": "check-1", "status": "paid"}}
                if path == "/checks/check-1":
                    return {"id": "check-1", "status": "paid", "total": 15000}
                if path == "/checks/check-1/financial-operations":
                    return [{"id": "operation-refund-1", "operation_type": "refund"}]
                if path.startswith("/orders/closed"):
                    return [{"id": "order-1", "status": "closed", "check": {"id": "check-1"}}]
                raise AssertionError(f"unexpected GET {path}")

        summary = {
            "restaurant_id": "restaurant-1",
            "node_device_id": "node-1",
            "manager_pin": "2222",
            "manager_employee_id": "manager-1",
            "hall_id": "hall-1",
            "table_ids": ["table-1"],
            "menu_item_ids": ["menu-normal"],
        }
        pos = RefundPOS()
        ctx = StackContext(FakeClient("cloud"), pos, FakeClient("license"))

        result = run_pos_refund_after_shift_close_suite(ctx, existing_summary=summary, client_device_id="client-1")

        self.assertEqual(result["name"], "pos_refund_after_shift_close")
        self.assertEqual(result["status"], "passed")
        self.assertEqual(result["details"]["original_employee_shift_status"], "closed")
        self.assertEqual(result["details"]["original_cash_shift_status"], "closed")
        refund_posts = [body for path, body, *_ in pos.posts if path == "/checks/check-1/refunds"]
        self.assertTrue(refund_posts[0]["command_id"].startswith("record-refund-after-shift-close-"))
        self.assertEqual(refund_posts[0]["approved_by_employee_id"], "manager-1")
        self.assertNotIn("session-manager-1", str(result))

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
