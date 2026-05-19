import pathlib
import sys
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "lib"))


class FakeClient:
    def __init__(self):
        self.posts = []
        self.gets = []
        self.counter = 0
        self.menu_attempts = 0

    def post(self, path, body=None, headers=None, expected_status=(200, 201)):
        self.posts.append((path, body or {}, headers or {}, tuple(expected_status)))
        self.counter += 1
        if path == "/auth/pin-login":
            return {
                "session": {"id": "session-1"},
                "actor": {"employee_id": "manager-1"},
            }
        if path.endswith("/master-data/publish"):
            return {"id": f"publication-{self.counter}"}
        if path.endswith("/devices/generate-pairing-code"):
            return {"pairing_code": "ABC23456", "node_device_id": body["node_device_id"]}
        if path == "/system/provisioning/pair-via-license":
            return {"paired": True, "node_device_id": "node-1", "restaurant_id": "restaurant-1"}
        if path == "/restaurants":
            return {"id": "restaurant-1"}
        if path == "/roles":
            return {"id": f"role-{self.counter}"}
        if path == "/employees":
            return {"id": "manager-1" if body.get("pin") == "2222" else "cashier-1"}
        if path == "/halls":
            return {"id": "hall-1"}
        if path == "/tables":
            return {"id": "table-1"}
        if path == "/catalog/items":
            return {"id": f"catalog-{self.counter}"}
        if path == "/menu/items":
            return {"id": "menu-service" if body.get("name", "").endswith("Service") else "menu-tea"}
        if path == "/master-data/modifiers/groups":
            return {"id": "modifier-group-1"}
        if path == "/master-data/modifiers/options":
            return {"id": "modifier-option-1"}
        if path == "/master-data/modifiers/bindings":
            return {"id": "modifier-binding-1"}
        return {"id": f"{path.strip('/').replace('/', '-')}-{self.counter}"}

    def get(self, path, headers=None, expected_status=(200,)):
        self.gets.append((path, headers or {}, tuple(expected_status)))
        if path == "/system/provisioning-status":
            return {"node_device_id": "node-1", "paired": False}
        if path == "/system/pairing-status":
            return {"paired": True}
        if path.startswith("/halls"):
            return [{"id": "hall-1"}]
        if path.startswith("/tables"):
            return [{"id": "table-1"}]
        if path == "/menu/items":
            self.menu_attempts += 1
            items = [
                {
                    "id": "menu-tea",
                    "modifier_groups": [
                        {"id": "modifier-group-1", "options": [{"id": "modifier-option-1"}]}
                    ],
                },
                {"id": "menu-service"},
            ]
            if self.menu_attempts >= 2:
                items.append({"id": "menu-sync"})
            return items
        if path == "/sync/status":
            return {"status": "ok"}
        if path.startswith("/sync/outbox") or path.startswith("/sync/local-events"):
            return []
        raise AssertionError(f"unexpected GET {path}")


class SeedWorkflowTest(unittest.TestCase):
    def test_create_cloud_seed_uses_cloud_master_data_apis_and_returns_summary(self):
        from mhpos_seed import create_cloud_seed

        client = FakeClient()
        summary = create_cloud_seed(
            client,
            restaurant_name="Demo",
            cashier_pin="1111",
            manager_pin="2222",
            node_device_id="node-1",
            suffix="unit",
        )

        paths = [path for path, *_ in client.posts]
        self.assertIn("/restaurants", paths)
        self.assertIn("/roles", paths)
        self.assertIn("/employees", paths)
        self.assertIn("/master-data/modifiers/groups", paths)
        self.assertIn("/restaurants/restaurant-1/master-data/publish", paths)
        self.assertEqual(summary["restaurant_id"], "restaurant-1")
        self.assertEqual(summary["node_device_id"], "node-1")
        self.assertEqual(len(summary["menu_item_ids"]), 3)

    def test_verify_pos_read_model_checks_login_menu_table_and_modifier(self):
        from mhpos_seed import verify_pos_read_model

        client = FakeClient()
        result = verify_pos_read_model(
            client,
            {
                "restaurant_id": "restaurant-1",
                "node_device_id": "node-1",
                "manager_pin": "2222",
                "manager_employee_id": "manager-1",
                "hall_id": "hall-1",
                "table_ids": ["table-1"],
                "menu_item_ids": ["menu-tea", "menu-service"],
                "modifier_option_id": "modifier-option-1",
            },
            client_device_id="client-1",
        )

        self.assertEqual(result["manager_employee_id"], "manager-1")
        self.assertEqual(result["menu_items_checked"], 2)

    def test_wait_for_menu_item_polls_until_synced_item_is_visible(self):
        from mhpos_seed import wait_for_menu_item

        client = FakeClient()
        found = wait_for_menu_item(
            client,
            menu_item_id="menu-sync",
            headers={"X-Session-ID": "session-1"},
            timeout_seconds=3,
            interval_seconds=0,
        )

        self.assertEqual(found["id"], "menu-sync")
        self.assertEqual(client.menu_attempts, 2)

    def test_redacted_summary_hides_local_demo_secrets(self):
        from mhpos_seed import redacted_summary

        safe = redacted_summary({"cashier_pin": "1111", "manager_pin": "2222", "pairing_code": "ABC23456"})

        self.assertEqual(safe["cashier_pin"], "<redacted>")
        self.assertEqual(safe["manager_pin"], "<redacted>")
        self.assertEqual(safe["pairing_code"], "<redacted>")


if __name__ == "__main__":
    unittest.main()
