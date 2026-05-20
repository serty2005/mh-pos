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
        if path in ("/roles", "/master-data/roles"):
            return {"id": f"role-{self.counter}"}
        if path in ("/employees", "/master-data/employees"):
            return {"id": "manager-1" if body.get("pin") == "2222" else "cashier-1"}
        if path in ("/halls", "/master-data/floor/halls"):
            return {"id": "hall-1"}
        if path in ("/tables", "/master-data/floor/tables"):
            return {"id": "table-1"}
        if path in ("/catalog/items", "/master-data/catalog/items"):
            return {"id": f"catalog-{self.counter}"}
        if path in ("/menu/items", "/master-data/menu/items"):
            return {"id": "menu-service" if body.get("name", "").endswith("Service") else "menu-tea"}
        if path == "/master-data/modifiers/groups":
            return {"id": "modifier-group-1"}
        if path == "/master-data/modifiers/options":
            return {"id": "modifier-option-1"}
        if path == "/master-data/modifiers/bindings":
            return {"id": "modifier-binding-1"}
        if path == "/master-data/publications":
            return {"id": f"publication-{self.counter}"}
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
        self.assertIn("/master-data/roles", paths)
        self.assertIn("/master-data/employees", paths)
        self.assertIn("/master-data/modifiers/groups", paths)
        self.assertIn("/master-data/publications", paths)
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




    def test_seed_reference_data_creates_catalog_halls_modifiers_and_publication(self):
        from mhpos_seed import DEFAULT_REFERENCE_SEED, seed_reference_data

        client = FakeClient()
        result = seed_reference_data(client, "restaurant-1", DEFAULT_REFERENCE_SEED, publication_node_device_id="node-1")

        self.assertTrue(result["catalog_item_ids"])
        self.assertTrue(result["menu_item_ids"])
        self.assertTrue(result["hall_ids"])
        self.assertTrue(result["table_ids"])
        self.assertTrue(result["modifier_group_ids"])
        self.assertTrue(result["modifier_option_ids"])
        self.assertEqual(result["publication_id"].startswith("publication-"), True)

    def test_generate_edge_sync_events_produces_requested_batches(self):
        from mhpos_seed import generate_edge_sync_events

        client = FakeClient()
        events = generate_edge_sync_events(client, {"restaurant_id": "restaurant-1", "node_device_id": "node-1"}, batches=2)

        self.assertEqual(len(events), 2)
        self.assertTrue(all(event["publication_id"].startswith("publication-") for event in events))

if __name__ == "__main__":
    unittest.main()
