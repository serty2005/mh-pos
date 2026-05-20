import pathlib
import sys
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "lib"))


class OpenAPIContractTest(unittest.TestCase):
    def test_default_contract_resolves_operation_to_request(self):
        from mhpos_contract import load_default_contract

        contract = load_default_contract()
        request = contract.build_request(
            "generatePairingCode",
            path_params={"restaurant_id": "restaurant 1"},
            body={"node_device_id": "node-1", "display_name": "POS", "expires_in_minutes": 30},
        )

        self.assertEqual(request["method"], "POST")
        self.assertEqual(request["path"], "/restaurants/restaurant%201/devices/generate-pairing-code")
        self.assertEqual(request["expected_status"], (201,))
        self.assertTrue(request["api_prefix"])

    def test_contract_rejects_missing_required_body_field(self):
        from mhpos_contract import load_default_contract

        contract = load_default_contract()

        with self.assertRaisesRegex(ValueError, "display_name"):
            contract.build_request(
                "registerCloudProvisioning",
                body={"cloud_url": "http://cloud-api:8090", "app_version": "test"},
            )

    def test_health_operation_is_root_path_without_api_prefix(self):
        from mhpos_contract import load_default_contract

        contract = load_default_contract()
        request = contract.build_request("health")

        self.assertEqual(request["method"], "GET")
        self.assertEqual(request["path"], "/health")
        self.assertFalse(request["api_prefix"])

    def test_license_pairing_operations_are_defined(self):
        from mhpos_contract import load_default_contract

        contract = load_default_contract()
        register = contract.build_request(
            "registerLicensePairingCode",
            body={
                "pairing_code": "123456",
                "cloud_url": "http://cloud-api:8090",
                "restaurant_id": "restaurant-1",
                "node_device_id": "node-1",
                "credentials": {"type": "node_token", "token": "secret"},
                "expires_at": "2026-05-20T10:00:00Z",
            },
        )
        resolve = contract.build_request(
            "resolveLicensePairingCode",
            body={"pairing_code": "123456", "node_device_id": "node-1"},
        )

        self.assertEqual(register["method"], "POST")
        self.assertEqual(register["path"], "/pairing-codes")
        self.assertEqual(resolve["method"], "POST")
        self.assertEqual(resolve["path"], "/pairing-codes/resolve")


if __name__ == "__main__":
    unittest.main()
