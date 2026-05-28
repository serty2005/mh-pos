import importlib.util
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
SCRIPT_PATH = ROOT / "scripts" / "seed-dev-system.py"


def load_seed_module():
    spec = importlib.util.spec_from_file_location("seed_dev_system", SCRIPT_PATH)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FakeClient:
    def __init__(self, name):
        self.name = name
        self.calls = []
        self.counter = 0
        self.full_smoke = False
        self.catalog_get_count = 0

    def root_get(self, path, expected_status=(200,)):
        self.calls.append(("GET", path, None, tuple(expected_status)))
        return {"status": "ok", "service": self.name}

    def request(self, method, path, body=None, expected_status=(200, 201), headers=None):
        self.calls.append((method, path, body or {}, tuple(expected_status)))
        self.counter += 1
        if path == "/api/v1/system/provisioning-status":
            return {"node_device_id": "edge-node-from-pos", "paired": False}
        if path == "/api/v1/restaurants":
            return {"id": "restaurant-1"}
        if path == "/api/v1/master-data/roles":
            return {"id": f"role-{self.counter}"}
        if path == "/api/v1/master-data/employees":
            return {"id": f"employee-{body['pin']}"}
        if path == "/api/v1/master-data/catalog/folders":
            return {"id": f"folder-{self.counter}"}
        if path == "/api/v1/master-data/catalog/folder-parameters":
            return {"id": f"folder-param-{self.counter}"}
        if path == "/api/v1/master-data/catalog/tags":
            return {"id": f"tag-{self.counter}"}
        if path == "/api/v1/master-data/catalog/items":
            return {"id": f"catalog-{self.counter}"}
        if path == "/api/v1/master-data/catalog/item-tags":
            return {"id": f"item-tag-{self.counter}"}
        if path == "/api/v1/master-data/menu/categories":
            return {"id": f"category-{self.counter}"}
        if path == "/api/v1/master-data/menu/items":
            return {"id": f"menu-{self.counter}"}
        if path == "/api/v1/master-data/modifiers/groups":
            return {"id": f"modifier-group-{self.counter}"}
        if path == "/api/v1/master-data/modifiers/options":
            return {"id": f"modifier-option-{self.counter}"}
        if path == "/api/v1/master-data/modifiers/bindings":
            return {"id": f"modifier-binding-{self.counter}"}
        if path == "/api/v1/master-data/pricing/policies":
            return {"id": f"pricing-policy-{self.counter}"}
        if path == "/api/v1/master-data/recipes/items":
            return {"id": f"recipe-{self.counter}"}
        if path == "/api/v1/master-data/inventory/stop-list":
            return {"id": f"stop-list-{self.counter}"}
        if path == "/api/v1/master-data/floor/halls":
            return {"id": f"hall-{self.counter}"}
        if path == "/api/v1/master-data/floor/tables":
            return {"id": f"table-{self.counter}"}
        if path == "/api/v1/master-data/publications":
            return {"id": "publication-1"}
        if path == "/api/v1/restaurants/restaurant-1/devices/generate-pairing-code":
            return {"pairing_code": "PAIR1234", "node_device_id": body["node_device_id"]}
        if path == "/api/v1/system/provisioning/pair-via-license":
            return {"paired": True, "node_device_id": "edge-node-from-pos", "restaurant_id": "restaurant-1"}
        if path == "/api/v1/auth/pin-login":
            return {"session": {"id": f"session-{body.get('pin', '1')}"}, "actor": {"employee_id": f"employee-{body.get('pin', '2222')}"}}
        if path == "/api/v1/employee-shifts/open":
            return {"id": f"shift-{self.counter}"}
        if path == "/api/v1/cash-shifts/open":
            return {"id": f"cash-{self.counter}"}
        if path == "/api/v1/employee-shifts/current":
            return {"id": "shift-current"}
        if path == "/api/v1/cash-shifts/current":
            return {"id": "cash-current"}
        if path.startswith("/api/v1/halls"):
            return [{"id": "hall-1"}]
        if path.startswith("/api/v1/tables"):
            return [{"id": "table-1"}]
        if path.startswith("/api/v1/kitchen/tickets/ticket-line-soup/"):
            status_by_action = {
                "accept": "accepted",
                "start": "in_progress",
                "ready": "ready",
                "serve": "served",
                "recall": "recall",
            }
            action = path.rsplit("/", 1)[-1]
            return {"id": "ticket-line-soup", "order_line_id": "line-soup", "status": status_by_action[action]}
        if path.startswith("/api/v1/kitchen/order-queue"):
            return {"orders": [{"id": "order-1", "kitchen_order_status": "queued", "tickets": [{"id": "ticket-line-soup", "order_line_id": "line-soup", "status": "new"}]}]}
        if path.startswith("/api/v1/kitchen/tickets"):
            return [{"id": "ticket-line-soup", "order_line_id": "line-soup", "status": "new"}]
        if path == "/api/v1/catalog/items":
            self.catalog_get_count += 1
            items = [
                {"id": "catalog-soup", "type": "dish", "name": "Tom Yum Soup"},
                {"id": "catalog-sirloin", "type": "good", "name": "Beef Sirloin"},
                {"id": "catalog-sauce", "type": "semi_finished", "name": "House Sauce"},
            ]
            if self.catalog_get_count > 1:
                items.append({"id": "catalog-smoke-herb", "type": "good", "name": "Smoke Herb"})
            return items
        if path == "/api/v1/kitchen/catalog/items/catalog-soup/recipe":
            return {
                "catalog_item": {"id": "catalog-soup"},
                "recipe_version": {"id": "recipe-version-soup"},
                "ingredients": [{"line_id": "recipe-line-1", "catalog_item_id": "catalog-sirloin"}],
            }
        if path == "/api/v1/menu/items":
            return [
                {
                    "id": "menu-soup",
                    "modifier_groups": [{
                        "id": "modifier-group-spice",
                        "required": True,
                        "min_count": 1,
                        "max_count": 1,
                        "options": [{"id": "modifier-option-mild"}],
                    }],
                },
                {"id": "menu-stopped", "modifier_groups": []},
            ]
        if path == "/api/v1/orders":
            return {"id": "order-1", "status": "open"}
        if path == "/api/v1/orders/order-1/lines":
            if body.get("menu_item_id") == "menu-stopped":
                return {"error_code": "SALE_ITEM_STOP_LISTED"}
            return {"id": "line-soup", "menu_item_id": body.get("menu_item_id"), "modifiers": body.get("selected_modifiers", [])}
        if path == "/api/v1/orders/order-1/precheck":
            return {"id": "precheck-1", "status": "issued", "total": 34900, "currency": "RUB"}
        if path == "/api/v1/prechecks/precheck-1/payments":
            return {"id": "payment-1", "precheck_id": "precheck-1", "amount": body.get("amount"), "status": "captured"}
        if path == "/api/v1/orders/order-1":
            return {"id": "order-1", "status": "closed", "check": {"id": "check-1", "status": "paid"}}
        if path.startswith("/api/v1/sync/edge-events"):
            if "event_type=ItemServed" in path:
                if self.full_smoke:
                    return [
                        {"event_id": "event-item-served-2", "event_type": "ItemServed", "aggregate_id": "ticket-line-soup"},
                        {"event_id": "event-item-served", "event_type": "ItemServed", "aggregate_id": "ticket-line-soup"},
                    ]
                return [{"event_id": "event-item-served", "event_type": "ItemServed", "aggregate_id": "ticket-line-soup"}]
            if "event_type=KitchenTicketStatusChanged" in path:
                return [
                    {"event_id": f"event-kds-status-{i}", "event_type": "KitchenTicketStatusChanged", "aggregate_id": "ticket-line-soup"}
                    for i in range(8)
                ]
            for event_type in ("StockReceiptCaptured", "InventoryCountCaptured", "StockWriteOffCaptured", "ProductionCompleted"):
                if f"event_type={event_type}" in path:
                    aggregate = {
                        "StockReceiptCaptured": "receipt-1",
                        "InventoryCountCaptured": "count-1",
                        "StockWriteOffCaptured": "writeoff-1",
                        "ProductionCompleted": "production-1",
                    }[event_type]
                    return [{"event_id": f"event-{event_type}", "event_type": event_type, "aggregate_id": aggregate}]
            return [{"event_id": "event-check-closed", "event_type": "CheckClosed", "aggregate_id": "check-1"}]
        if path.startswith("/api/v1/olap/raw-business-events"):
            if "event_type=ItemServed" in path:
                return [{"event_id": "event-item-served-2"}, {"event_id": "event-item-served"}]
            if "event_type=KitchenTicketStatusChanged" in path:
                return [{"event_id": "event-kds-status-0"}, {"event_id": "event-kds-status-1"}]
            return []
        if path.startswith("/api/v1/inventory/stock-ledger"):
            if "source_event_type=CheckClosed" in path:
                return []
            if "source_event_type=Stock" in path or "source_event_type=InventoryCountCaptured" in path or "source_event_type=ProductionCompleted" in path:
                return [{"id": "ledger-stock-1", "source_event_id": "event-stock"}]
            return [
                {"id": "ledger-1", "source_event_id": "event-item-served", "source_event_type": "ItemServed", "order_line_id": "line-soup", "catalog_item_id": "catalog-sirloin"},
                {"id": "ledger-2", "source_event_id": "event-item-served", "source_event_type": "ItemServed", "order_line_id": "line-soup", "catalog_item_id": "catalog-sauce"},
            ]
        if path == "/api/v1/kitchen/stock-receipts":
            return {"id": body["receipt_id"], "warehouse_id": body.get("warehouse_id", ""), "event_type": "StockReceiptCaptured"}
        if path == "/api/v1/kitchen/inventory-counts":
            return {"id": body["count_id"], "warehouse_id": body.get("warehouse_id", ""), "event_type": "InventoryCountCaptured"}
        if path == "/api/v1/kitchen/stock-write-offs":
            return {"id": body["write_off_id"], "warehouse_id": body.get("warehouse_id", ""), "event_type": "StockWriteOffCaptured"}
        if path == "/api/v1/kitchen/productions":
            return {"id": body["production_id"], "warehouse_id": body.get("warehouse_id", ""), "event_type": "ProductionCompleted"}
        if path == "/api/v1/kitchen/catalog-suggestions":
            return {"id": body["suggestion_id"], "kind": "catalog", "status": "pending_sync"}
        if path == "/api/v1/kitchen/recipe-suggestions":
            return {"id": body["suggestion_id"], "kind": "recipe", "status": "pending_sync"}
        if path.startswith("/api/v1/master-data/catalog-suggestions"):
            if path.endswith("/approve"):
                return {"id": "cloud-catalog-suggestion-1", "suggestion_id": "catalog-suggestion", "status": "approved"}
            return [{"id": "cloud-catalog-suggestion-1", "suggestion_id": "catalog-suggestion", "status": "pending"}]
        if path.startswith("/api/v1/master-data/recipe-suggestions"):
            if path.endswith("/approve"):
                return {"id": "cloud-recipe-suggestion-1", "suggestion_id": "recipe-suggestion", "status": "approved"}
            return [{"id": "cloud-recipe-suggestion-1", "suggestion_id": "recipe-suggestion", "status": "pending"}]
        if path.startswith("/api/v1/kitchen/proposals"):
            return [
                {"id": "catalog-suggestion", "kind": "catalog", "status": "approved"},
                {"id": "recipe-suggestion", "kind": "recipe", "status": "approved"},
            ]
        if path == "/api/v1/sync/status":
            return {"status": "ok"}
        raise AssertionError(f"unexpected request {method} {path}")


class SeedDevSystemTest(unittest.TestCase):
    def test_scripts_has_one_user_facing_python_entrypoint(self):
        names = sorted(path.name for path in (ROOT / "scripts").glob("*.py"))

        self.assertEqual(names, ["seed-dev-system.py"])

    def test_default_output_is_next_to_seed_script(self):
        module = load_seed_module()
        output = pathlib.Path(module.parse_args([]).output)

        self.assertTrue(output.is_absolute())
        self.assertEqual(output.parent, ROOT / "scripts")
        self.assertEqual(output.name, ".seed-dev-system-summary.json")

    def test_seed_dataset_contains_user_data_without_preassigned_ids(self):
        module = load_seed_module()

        dataset = module.build_seed_dataset("unit")

        forbidden_suffixes = ("id", "_id", "pairing_code", "node_device_id")
        flattened = repr(dataset)
        for key in forbidden_suffixes:
            self.assertNotIn(f"'{key}':", flattened)
            self.assertNotIn(f'"{key}":', flattened)
        self.assertGreaterEqual(len(dataset["roles"]), 6)
        self.assertGreaterEqual(len(dataset["employees"]), 6)
        self.assertGreaterEqual(len(dataset["catalog_items"]), 6)
        self.assertTrue(dataset["pricing_policies"])
        self.assertTrue(dataset["recipes"])
        self.assertTrue(dataset["stop_list"])

    def test_seed_full_system_generates_pairing_after_all_master_data(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        pos = FakeClient("pos")
        license_client = FakeClient("license")

        summary = module.seed_full_system(
            cloud,
            pos,
            license_client,
            cloud_base_url="http://cloud-api:8090",
            client_device_id="unit-client",
            suffix="unit",
            wait_seconds=1,
            interval_seconds=0,
        )

        cloud_paths = [path for _, path, _, _ in cloud.calls]
        pairing_index = cloud_paths.index("/api/v1/restaurants/restaurant-1/devices/generate-pairing-code")
        publication_index = cloud_paths.index("/api/v1/master-data/publications")
        self.assertGreater(pairing_index, publication_index)
        self.assertEqual(summary["node_device_id"], "edge-node-from-pos")
        self.assertEqual(summary["pairing_code"], "PAIR1234")
        self.assertIn("waiter_pin", summary["pins"])
        self.assertIn("kitchen_pin", summary["pins"])
        self.assertIn("support_pin", summary["pins"])

    def test_minimal_flow_smoke_runs_waiter_to_cloud_inventory_ledger(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        pos = FakeClient("pos")

        result = module.run_minimal_flow_smoke(
            cloud,
            pos,
            restaurant_id="restaurant-1",
            node_device_id="edge-node-from-pos",
            client_device_id="unit-client",
            pins={"waiter_pin": "3333", "cashier_pin": "1111", "kitchen_pin": "5555"},
            table_ids=["table-1"],
            menu_refs={"soup": "menu-soup", "sold_out_dessert": "menu-stopped"},
            catalog_refs={"sirloin": "catalog-sirloin", "sauce": "catalog-sauce"},
            wait_seconds=1,
            interval_seconds=0,
        )

        self.assertEqual(result["check_id"], "check-1")
        self.assertEqual(result["kitchen_ticket_id"], "ticket-line-soup")
        self.assertEqual(result["item_served_event_id"], "event-item-served")
        self.assertEqual(result["check_closed_event_id"], "event-check-closed")
        self.assertEqual(result["served_ledger_entry_count"], 2)
        self.assertEqual(result["check_closed_delta_entry_count"], 0)
        self.assertEqual(result["ledger_catalog_item_ids"], ["catalog-sauce", "catalog-sirloin"])
        self.assertEqual(result["blocked_sale_error_code"], "SALE_ITEM_STOP_LISTED")
        cloud_paths = [path for _, path, _, _ in cloud.calls]
        self.assertTrue(any("event_type=ItemServed" in path for path in cloud_paths))
        self.assertTrue(any("event_type=CheckClosed" in path for path in cloud_paths))
        self.assertTrue(any("source_event_type=ItemServed" in path for path in cloud_paths))
        self.assertTrue(any("source_event_type=CheckClosed" in path for path in cloud_paths))

    def test_kitchen_process_smoke_covers_kds_stock_olap_and_proposals(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        cloud.full_smoke = True
        pos = FakeClient("pos")

        result = module.run_kitchen_process_smoke_flow(
            cloud,
            pos,
            restaurant_id="restaurant-1",
            node_device_id="edge-node-from-pos",
            client_device_id="unit-client",
            pins={"waiter_pin": "3333", "kitchen_pin": "5555"},
            table_ids=["table-1"],
            menu_refs={"soup": "menu-soup"},
            catalog_refs={"soup": "catalog-soup", "sirloin": "catalog-sirloin", "sauce": "catalog-sauce"},
            wait_seconds=1,
            interval_seconds=0,
        )

        self.assertEqual(result["kitchen_ticket_id"], "ticket-line-soup")
        self.assertEqual(result["latest_item_served_event_id"], "event-item-served-2")
        self.assertEqual(result["olap_item_served_event_count"], 2)
        self.assertEqual(result["olap_status_event_count"], 2)
        self.assertEqual(set(result["stock"].keys()), {"receipt", "count", "write_off", "production"})
        self.assertEqual(result["approved_catalog_status"], "approved")
        self.assertEqual(result["approved_recipe_status"], "approved")
        self.assertGreaterEqual(result["edge_catalog_item_count_after"], result["edge_catalog_item_count_before"] + 1)


if __name__ == "__main__":
    unittest.main()
