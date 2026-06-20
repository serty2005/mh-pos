import importlib.util
import ast
import contextlib
import io
import json
import pathlib
import unittest
from unittest import mock


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
            catalog_ids = {
                "Tom Yum Soup": "catalog-soup",
                "Beef Sirloin": "catalog-sirloin",
                "House Sauce": "catalog-sauce",
                "Sold Out Cheesecake": "catalog-stopped",
            }
            return {"id": catalog_ids.get(body.get("name"), f"catalog-{self.counter}")}
        if path == "/api/v1/master-data/catalog/item-tags":
            return {"id": f"item-tag-{self.counter}"}
        if path == "/api/v1/master-data/menu/categories":
            return {"id": f"category-{self.counter}"}
        if path == "/api/v1/master-data/menu/items":
            menu_ids = {
                "Tom Yum Soup": "menu-soup",
                "Sold Out Cheesecake": "menu-stopped",
            }
            return {"id": menu_ids.get(body.get("name"), f"menu-{self.counter}")}
        if path == "/api/v1/master-data/modifiers/groups":
            return {"id": f"modifier-group-{self.counter}"}
        if path == "/api/v1/master-data/modifiers/options":
            return {"id": f"modifier-option-{self.counter}"}
        if path == "/api/v1/master-data/modifiers/bindings":
            return {"id": f"modifier-binding-{self.counter}"}
        if path == "/api/v1/master-data/pricing/policies":
            return {"id": f"pricing-policy-{self.counter}"}
        if path == "/api/v1/master-data/recipes/versions/drafts":
            owner = body["owner_catalog_item_id"]
            lines = [
                {"id": f"recipe-line-{owner}-{index + 1}", **line}
                for index, line in enumerate(body.get("lines", []))
            ]
            return {"version": {"id": f"recipe-version-{owner}", "status": "draft"}, "lines": lines}
        if path.startswith("/api/v1/master-data/recipes/versions/") and path.endswith("/submit"):
            version_id = path.split("/")[-2]
            return {"id": f"recipe-suggestion-{version_id}", "recipe_version_id": version_id, "status": "pending"}
        if path == "/api/v1/master-data/inventory/stop-list":
            return {"id": f"stop-list-{self.counter}"}
        if path == "/api/v1/master-data/floor/halls":
            return {"id": f"hall-{self.counter}"}
        if path == "/api/v1/master-data/floor/tables":
            return {"id": f"table-{self.counter}"}
        if path == "/api/v1/master-data/publications":
            return {"id": "publication-1"}
        if path == "/api/v1/restaurants/restaurant-1/devices/generate-pairing-code":
            if "node_device_id" in body:
                raise AssertionError("Cloud pairing code generation must not receive edge node_device_id")
            return {"pairing_code": "PAIR1234", "pairing_id": "pairing-1"}
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
                return {"error": {"code": "SALE_STOP_LIST_CONFLICT", "message_key": "errors.stopListConflict"}}
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
            if "event_type=CheckClosed" in path:
                return [{"event_id": "event-check-closed"}]
            return []
        if path.startswith("/api/v1/olap/stock-moves"):
            if "source_event_type=ItemServed" in path:
                return [
                    {"ledger_entry_id": "olap-ledger-1", "source_event_id": "event-item-served-2", "source_event_type": "ItemServed"},
                    {"ledger_entry_id": "olap-ledger-minimal", "source_event_id": "event-item-served", "source_event_type": "ItemServed"},
                ]
            for event_type in ("StockReceiptCaptured", "InventoryCountCaptured", "StockWriteOffCaptured", "ProductionCompleted"):
                if f"source_event_type={event_type}" in path:
                    return [{"ledger_entry_id": f"olap-ledger-{event_type}", "source_event_id": f"event-{event_type}", "source_event_type": event_type}]
            return []
        if path.startswith("/api/v1/olap/stock-move-summary"):
            return [
                {"group_by": "catalog_item", "group_key": "catalog-sirloin", "catalog_item_id": "catalog-sirloin", "move_count": 1},
                {"group_by": "catalog_item", "group_key": "catalog-sauce", "catalog_item_id": "catalog-sauce", "move_count": 1},
            ]
        if path.startswith("/api/v1/olap/sales-kitchen-summary"):
            return [
                {"group_by": "event_type", "group_key": "ItemServed", "event_type": "ItemServed", "event_count": 1},
                {"group_by": "event_type", "group_key": "CheckClosed", "event_type": "CheckClosed", "event_count": 1},
            ]
        if path.startswith("/api/v1/olap/kitchen-timing-summary"):
            return [{"group_by": "business_date", "group_key": "2026-06-14", "ticket_count": 1}]
        if path.startswith("/api/v1/inventory/stock-ledger"):
            if "source_event_type=CheckClosed" in path:
                return []
            if "source_event_type=Stock" in path or "source_event_type=InventoryCountCaptured" in path or "source_event_type=ProductionCompleted" in path:
                return [{"id": "ledger-stock-1", "source_event_id": "event-stock"}]
            return [
                {"id": "ledger-1", "source_event_id": "event-item-served", "source_event_type": "ItemServed", "order_line_id": "line-soup", "catalog_item_id": "catalog-sirloin"},
                {"id": "ledger-2", "source_event_id": "event-item-served", "source_event_type": "ItemServed", "order_line_id": "line-soup", "catalog_item_id": "catalog-sauce"},
            ]
        if path.startswith("/api/v1/inventory/stock-balances"):
            return [
                {"catalog_item_id": "catalog-sirloin", "warehouse_id": "warehouse-main", "quantity_on_hand": "-0.120", "unit_code": "g", "costing_status": "estimated", "needs_recalculation": True},
                {"catalog_item_id": "catalog-sauce", "warehouse_id": "warehouse-main", "quantity_on_hand": "-0.030", "unit_code": "g", "costing_status": "estimated", "needs_recalculation": True},
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
                return {"id": "cloud-catalog-suggestion-1", "suggestion_id": "catalog-suggestion", "status": "approved", "applied_catalog_item_id": "catalog-smoke-herb"}
            return [{"id": "cloud-catalog-suggestion-1", "suggestion_id": "catalog-suggestion", "status": "pending"}]
        if path.startswith("/api/v1/master-data/recipe-suggestions"):
            if path.endswith("/approve"):
                return {"id": path.split("/")[-2], "suggestion_id": "recipe-suggestion", "status": "approved"}
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

    def test_json_client_disables_system_proxy_for_local_stack(self):
        module = load_seed_module()

        with (
            mock.patch.object(module.urllib.request, "ProxyHandler", return_value="proxy-handler") as proxy_handler,
            mock.patch.object(module.urllib.request, "build_opener", return_value="opener") as build_opener,
        ):
            client = module.JsonClient("http://localhost:8090")

        self.assertEqual(client.opener, "opener")
        proxy_handler.assert_called_once_with({})
        build_opener.assert_called_once_with("proxy-handler")

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
        menu_items = [item for item in dataset["catalog_items"] if "price_minor" in item]
        self.assertTrue(menu_items)
        self.assertTrue(all(item.get("category_ref") for item in menu_items))
        self.assertTrue(all(item.get("tags") for item in menu_items))
        self.assertTrue(dataset["pricing_policies"])
        self.assertTrue(dataset["recipes"])
        self.assertTrue(dataset["stop_list"])

    def test_cloud_owned_seed_extension_plan_matches_dataset(self):
        module = load_seed_module()
        dataset = module.build_seed_dataset("unit")

        module.validate_seed_extension_plan(dataset)
        dataset_keys = {item["dataset_key"] for item in module.CLOUD_OWNED_SEED_SURFACES}
        publication_streams = {item["publication_stream"] for item in module.CLOUD_OWNED_SEED_SURFACES}
        read_checks = {item["pos_read_check"] for item in module.CLOUD_OWNED_SEED_SURFACES}

        for key in (
            "restaurant",
            "roles",
            "employees",
            "catalog_items",
            "menu_categories",
            "recipes",
            "stop_list",
            "floor",
        ):
            self.assertIn(key, dataset_keys)
        for stream in ("restaurants", "staff", "catalog", "menu", "pricing_policy", "recipes", "inventory_reference", "floor"):
            self.assertIn(stream, publication_streams)
        for read_check in ("pin_login", "menu_items", "kitchen_recipe", "blocked_sale"):
            self.assertIn(read_check, read_checks)

    def test_seed_dataset_contains_kitchen_role_pin_and_smoke_permissions(self):
        module = load_seed_module()

        dataset = module.build_seed_dataset("unit")
        kitchen_role = next(role for role in dataset["roles"] if role["ref"] == "kitchen")
        kitchen_employee = next(employee for employee in dataset["employees"] if employee["pin_name"] == "kitchen_pin")
        permissions = json.loads(module.permissions_json(kitchen_role["profile"]))

        self.assertEqual(kitchen_employee["role_ref"], "kitchen")
        for permission in (
            "pos.catalog.view",
            "pos.kitchen.view",
            "pos.kitchen.status.change",
            "pos.kitchen.catalog.view",
            "pos.kitchen.recipe.view",
            "pos.kitchen.recipe.suggest",
            "pos.kitchen.catalog.suggest",
            "pos.kitchen.stock.receipt",
            "pos.kitchen.stock.inventory_count",
            "pos.kitchen.stock.write_off",
            "pos.kitchen.production.complete",
        ):
            self.assertTrue(permissions.get(permission), f"missing kitchen smoke permission {permission}")

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
        self.assertEqual(summary["pairing_id"], "pairing-1")
        pairing_call = cloud.calls[pairing_index]
        self.assertEqual(pairing_call[2], {"display_name": "POS Terminal unit", "expires_in_minutes": 30})
        self.assertIn("/api/v1/system/provisioning/pair-via-license", [path for _, path, _, _ in pos.calls])
        self.assertEqual([path for _, path, _, _ in license_client.calls], ["/health"])
        self.assertIn("waiter_pin", summary["pins"])
        self.assertIn("kitchen_pin", summary["pins"])
        self.assertIn("support_pin", summary["pins"])

    def test_seed_script_uses_http_only_and_no_db_client_imports(self):
        tree = ast.parse(SCRIPT_PATH.read_text(encoding="utf-8"))
        imported = set()
        for node in ast.walk(tree):
            if isinstance(node, ast.Import):
                imported.update(alias.name.split(".", 1)[0] for alias in node.names)
            if isinstance(node, ast.ImportFrom) and node.module:
                imported.add(node.module.split(".", 1)[0])

        forbidden = {"sqlite3", "psycopg", "psycopg2", "asyncpg", "pgx", "clickhouse_connect", "clickhouse_driver", "subprocess"}
        self.assertFalse(imported & forbidden)

    def test_request_rejects_destructive_storage_archive_routes(self):
        module = load_seed_module()

        class Client:
            def request(self, method, path, body=None, expected_status=(200, 201), headers=None):
                raise AssertionError("guard must reject before HTTP call")

        for path in (
            "/api/v1/storage/archive/apply-plan",
            "/api/v1/storage/reset",
            "/api/v1/archives/apply",
        ):
            with self.assertRaises(RuntimeError):
                module.request(Client(), "POST", path, {})

    def test_financial_mutation_request_is_single_shot(self):
        module = load_seed_module()

        class FailingClient:
            def __init__(self):
                self.count = 0

            def request(self, method, path, body=None, expected_status=(200, 201), headers=None):
                self.count += 1
                raise RuntimeError("payment failed")

        client = FailingClient()
        with self.assertRaises(RuntimeError):
            module.request(client, "POST", "/api/v1/prechecks/precheck-1/payments", {"amount": 100})
        self.assertEqual(client.count, 1)

    def test_main_runs_both_smoke_flags_and_writes_separate_summary_sections(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        cloud.full_smoke = True
        pos = FakeClient("pos")
        license_client = FakeClient("license")
        clients = {
            "http://cloud.local": cloud,
            "http://pos.local": pos,
            "http://license.local": license_client,
        }
        module.JsonClient = lambda base_url: clients[base_url]

        with contextlib.redirect_stdout(io.StringIO()) as stdout:
            exit_code = module.main([
                "--cloud-base", "http://cloud.local",
                "--pos-base", "http://pos.local",
                "--license-base", "http://license.local",
                "--client-device-id", "unit-client",
                "--suffix", "unit",
                "--wait-seconds", "1",
                "--interval-seconds", "0",
                "--output", "",
                "--run-minimal-flow",
                "--run-kitchen-process-smoke",
            ])
        summary = json.loads(stdout.getvalue())

        self.assertEqual(exit_code, 0)
        self.assertIn("minimal_flow", summary)
        self.assertIn("kitchen_process_smoke", summary)
        self.assertEqual(summary["minimal_flow"]["check_id"], "check-1")
        self.assertEqual(summary["kitchen_process_smoke"]["kitchen_ticket_id"], "ticket-line-soup")

    def test_full_smoke_never_calls_destructive_storage_or_archive_routes(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        cloud.full_smoke = True
        pos = FakeClient("pos")
        license_client = FakeClient("license")

        module.seed_full_system(
            cloud,
            pos,
            license_client,
            cloud_base_url="http://cloud-api:8090",
            client_device_id="unit-client",
            suffix="unit",
            wait_seconds=1,
            interval_seconds=0,
            run_minimal_flow=True,
            run_kitchen_process_smoke=True,
        )

        forbidden = tuple(module.FORBIDDEN_MUTATING_ROUTE_FRAGMENTS)
        for client in (cloud, pos, license_client):
            for method, path, _, _ in client.calls:
                if method in ("POST", "PUT", "PATCH", "DELETE"):
                    self.assertFalse(any(fragment in path.lower() for fragment in forbidden), path)

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
        self.assertEqual(result["stock_balance_catalog_item_ids"], ["catalog-sauce", "catalog-sirloin"])
        self.assertEqual(result["stock_balance_count"], 2)
        self.assertEqual(result["olap_item_served_event_count"], 1)
        self.assertEqual(result["olap_check_closed_event_count"], 1)
        self.assertEqual(result["olap_item_served_stock_move_count"], 1)
        self.assertEqual(result["olap_stock_move_summary_count"], 2)
        self.assertEqual(result["olap_sales_kitchen_summary_group_keys"], ["CheckClosed", "ItemServed"])
        self.assertEqual(result["olap_kitchen_timing_summary_count"], 1)
        self.assertEqual(result["blocked_sale_error_code"], "SALE_STOP_LIST_CONFLICT")
        cloud_paths = [path for _, path, _, _ in cloud.calls]
        self.assertTrue(any("event_type=ItemServed" in path for path in cloud_paths))
        self.assertTrue(any("event_type=CheckClosed" in path for path in cloud_paths))
        self.assertTrue(any("source_event_type=ItemServed" in path for path in cloud_paths))
        self.assertTrue(any("source_event_type=CheckClosed" in path for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/inventory/stock-balances?") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/olap/stock-move-summary?") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/olap/sales-kitchen-summary?") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/olap/kitchen-timing-summary?") for path in cloud_paths))

    def test_seed_full_system_uses_manager_recipe_version_review_flow(self):
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
        self.assertTrue(summary["recipe_version_ids"])
        self.assertTrue(summary["recipe_line_ids"])
        self.assertTrue(summary["recipe_suggestion_ids"])
        self.assertIn("/api/v1/master-data/recipes/versions/drafts", cloud_paths)
        self.assertTrue(any(path.startswith("/api/v1/master-data/recipes/versions/") and path.endswith("/submit") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/master-data/recipe-suggestions/") and path.endswith("/approve") for path in cloud_paths))
        self.assertNotIn("/api/v1/master-data/recipes/items", cloud_paths)

    def test_minimal_flow_smoke_does_not_require_kitchen_pin(self):
        module = load_seed_module()
        cloud = FakeClient("cloud")
        pos = FakeClient("pos")

        result = module.run_minimal_flow_smoke(
            cloud,
            pos,
            restaurant_id="restaurant-1",
            node_device_id="edge-node-from-pos",
            client_device_id="unit-client",
            pins={"waiter_pin": "3333", "cashier_pin": "1111"},
            table_ids=["table-1"],
            menu_refs={"soup": "menu-soup", "sold_out_dessert": "menu-stopped"},
            catalog_refs={"sirloin": "catalog-sirloin", "sauce": "catalog-sauce"},
            wait_seconds=1,
            interval_seconds=0,
        )

        self.assertEqual(result["check_id"], "check-1")
        login_pins = [body["pin"] for _, path, body, _ in pos.calls if path == "/api/v1/auth/pin-login"]
        self.assertIn("1111", login_pins)

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
        self.assertGreaterEqual(result["latest_item_served_olap_stock_move_count"], 1)
        self.assertEqual(set(result["stock"].keys()), {"receipt", "count", "write_off", "production"})
        self.assertTrue(all(item["olap_stock_move_count"] >= 1 for item in result["stock"].values()))
        self.assertEqual(result["approved_catalog_status"], "approved")
        self.assertEqual(result["approved_recipe_status"], "approved")
        self.assertGreaterEqual(result["edge_catalog_item_count_after"], result["edge_catalog_item_count_before"] + 1)
        pos_paths = [path for _, path, _, _ in pos.calls]
        cloud_paths = [path for _, path, _, _ in cloud.calls]
        self.assertIn("/api/v1/kitchen/order-queue?limit=100&offset=0", pos_paths)
        for action in ("accept", "start", "ready", "serve", "recall"):
            self.assertIn(f"/api/v1/kitchen/tickets/ticket-line-soup/{action}", pos_paths)
        for path in (
            "/api/v1/kitchen/stock-receipts",
            "/api/v1/kitchen/inventory-counts",
            "/api/v1/kitchen/stock-write-offs",
            "/api/v1/kitchen/productions",
            "/api/v1/kitchen/catalog-suggestions",
            "/api/v1/kitchen/recipe-suggestions",
            "/api/v1/kitchen/proposals?kind=catalog&status=approved&limit=100&offset=0",
            "/api/v1/kitchen/proposals?kind=recipe&status=approved&limit=100&offset=0",
        ):
            self.assertIn(path, pos_paths)
        self.assertTrue(any(path.startswith("/api/v1/olap/raw-business-events?") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/olap/stock-moves?") for path in cloud_paths))
        self.assertTrue(any(path.startswith("/api/v1/inventory/stock-ledger?") for path in cloud_paths))
        self.assertTrue(any(path.endswith("/approve") for path in cloud_paths))


if __name__ == "__main__":
    unittest.main()
