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
            return {"session": {"id": "session-1"}, "actor": {"employee_id": "employee-2222"}}
        if path.startswith("/api/v1/halls"):
            return [{"id": "hall-1"}]
        if path.startswith("/api/v1/tables"):
            return [{"id": "table-1"}]
        if path == "/api/v1/menu/items":
            return [{"id": "menu-1", "modifier_groups": [{"options": [{"id": "modifier-option-1"}]}]}]
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
        self.assertGreaterEqual(len(dataset["roles"]), 5)
        self.assertGreaterEqual(len(dataset["employees"]), 5)
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
        self.assertIn("support_pin", summary["pins"])


if __name__ == "__main__":
    unittest.main()
