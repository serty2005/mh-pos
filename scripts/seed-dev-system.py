#!/usr/bin/env python3
import argparse
import json
import pathlib
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


API_PREFIX = "/api/v1"
SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
DEFAULT_OUTPUT = str(SCRIPT_DIR / ".seed-dev-system-summary.json")

PERMISSIONS = {
    "cashier": [
        "pos.employee_shift.open",
        "pos.employee_shift.close",
        "pos.employee_shift.view_current",
        "pos.employee_shift.recent",
        "pos.cash_session.open",
        "pos.cash_session.view_current",
        "pos.catalog.view",
        "pos.floor.view",
        "pos.menu.view",
        "pos.order.create",
        "pos.order.view",
        "pos.order.add_line",
        "pos.order.change_quantity",
        "pos.order.void_line",
        "pos.order.close",
        "pos.pricing.view",
        "pos.pricing.discount.apply",
        "pos.pricing.surcharge.apply",
        "pos.precheck.issue",
        "pos.precheck.view",
        "pos.precheck.reprint",
        "pos.payment.cash",
        "pos.payment.card.manual",
        "pos.check.view",
    ],
    "senior_cashier": [
        "pos.employee_shift.open",
        "pos.employee_shift.close",
        "pos.employee_shift.view_current",
        "pos.employee_shift.recent",
        "pos.cash_session.open",
        "pos.cash_session.close",
        "pos.cash_session.view_current",
        "pos.catalog.view",
        "pos.floor.view",
        "pos.menu.view",
        "pos.order.create",
        "pos.order.view",
        "pos.order.add_line",
        "pos.order.change_quantity",
        "pos.order.void_line",
        "pos.order.close",
        "pos.pricing.view",
        "pos.pricing.discount.apply",
        "pos.pricing.surcharge.apply",
        "pos.precheck.issue",
        "pos.precheck.view",
        "pos.precheck.reprint",
        "pos.precheck.cancel.request",
        "pos.payment.cash",
        "pos.payment.card.manual",
        "pos.payment.refund",
        "pos.check.view",
        "pos.sync.view",
    ],
    "waiter": [
        "pos.employee_shift.open",
        "pos.employee_shift.close",
        "pos.employee_shift.view_current",
        "pos.employee_shift.recent",
        "pos.catalog.view",
        "pos.floor.view",
        "pos.menu.view",
        "pos.order.create",
        "pos.order.view",
        "pos.order.add_line",
        "pos.order.change_quantity",
        "pos.order.void_line",
        "pos.order.close",
        "pos.pricing.view",
        "pos.precheck.issue",
        "pos.precheck.view",
        "pos.precheck.reprint",
        "pos.check.view",
    ],
    "manager": [
        "pos.employee_shift.open",
        "pos.employee_shift.close",
        "pos.employee_shift.view_current",
        "pos.employee_shift.recent",
        "pos.cash_session.open",
        "pos.cash_session.close",
        "pos.cash_session.view_current",
        "pos.cash_drawer.record_event",
        "pos.catalog.view",
        "pos.floor.view",
        "pos.menu.view",
        "pos.order.create",
        "pos.order.view",
        "pos.order.add_line",
        "pos.order.change_quantity",
        "pos.order.void_line",
        "pos.order.close",
        "pos.pricing.view",
        "pos.pricing.discount.apply",
        "pos.pricing.surcharge.apply",
        "pos.precheck.issue",
        "pos.precheck.view",
        "pos.precheck.reprint",
        "pos.precheck.cancel.request",
        "pos.precheck.cancel",
        "pos.payment.cash",
        "pos.payment.card.manual",
        "pos.payment.other",
        "pos.payment.refund",
        "pos.check.view",
        "pos.check.reprint",
        "pos.sync.view",
        "pos.sync.retry_failed",
    ],
    "kitchen": [
        "pos.employee_shift.view_current",
        "pos.kitchen.view",
        "pos.kitchen.status.change",
    ],
    "support_admin": [
        "pos.sync.view",
        "pos.sync.retry_failed",
    ],
}


class JsonClient:
    def __init__(self, base_url, timeout=20):
        self.base_url = normalize_base_url(base_url)
        self.timeout = timeout
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

    def root_get(self, path, expected_status=(200,)):
        return self._send("GET", path, None, expected_status)

    def request(self, method, path, body=None, expected_status=(200, 201), headers=None):
        return self._send(method, path, body, expected_status, headers=headers)

    def _send(self, method, path, body, expected_status, headers=None):
        url = self.base_url + path
        payload = None
        request_headers = {"Accept": "application/json"}
        request_headers.update(headers or {})
        if body is not None:
            payload = json.dumps(body).encode("utf-8")
            request_headers["Content-Type"] = "application/json"
        request = urllib.request.Request(url, data=payload, headers=request_headers, method=method)
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                data = response.read()
                status = response.status
        except urllib.error.HTTPError as exc:
            data = exc.read()
            status = exc.code
            if status not in expected_status:
                raise RuntimeError(f"{method} {url} returned HTTP {status}: {decode_body(data)}") from exc
        if status not in expected_status:
            raise RuntimeError(f"{method} {url} returned HTTP {status}: {decode_body(data)}")
        if not data:
            return {}
        try:
            return json.loads(data.decode("utf-8"))
        except json.JSONDecodeError:
            return {"raw": data.decode("utf-8", errors="replace")}


def normalize_base_url(value):
    text = str(value or "").strip().rstrip("/")
    if not text:
        raise ValueError("base URL is required")
    return text


def decode_body(data):
    return data.decode("utf-8", errors="replace") if data else ""


def request(client, method, path, body=None, expected_status=(200, 201), query=None, headers=None):
    if query:
        path = path + "?" + urllib.parse.urlencode({k: v for k, v in query.items() if v not in (None, "")})
    if hasattr(client, "request"):
        try:
            return client.request(method, path, body, expected_status=expected_status, headers=headers)
        except TypeError:
            return client.request(method, path, body, expected_status=expected_status)
    raise TypeError("client must expose request(method, path, body, expected_status)")


def root_health(client):
    if hasattr(client, "root_get"):
        return client.root_get("/health", expected_status=(200,))
    return request(client, "GET", "/health", expected_status=(200,))


def permissions_json(profile):
    values = PERMISSIONS[profile]
    return json.dumps({item: True for item in sorted(set(values))}, separators=(",", ":"))


def command_suffix():
    return uuid.uuid4().hex[:8]


def slug(value):
    text = re.sub(r"[^A-Za-z0-9]+", "-", value.upper()).strip("-")
    return text or "ITEM"


def system_sku(kind, name, suffix):
    return f"SEED-{kind.upper()}-{slug(name)}-{suffix}"


def build_seed_dataset(suffix):
    return {
        "restaurant": {
            "name": f"MH Demo Restaurant {suffix}",
            "timezone": "Europe/Moscow",
            "currency": "RUB",
            "business_day_mode": "standard",
            "business_day_boundary_local_time": "04:00",
        },
        "roles": [
            {"ref": "cashier", "name": "Cashier", "profile": "cashier"},
            {"ref": "senior_cashier", "name": "Senior Cashier", "profile": "senior_cashier"},
            {"ref": "waiter", "name": "Waiter", "profile": "waiter"},
            {"ref": "manager", "name": "Manager", "profile": "manager"},
            {"ref": "kitchen", "name": "Kitchen Display", "profile": "kitchen"},
            {"ref": "support", "name": "Support Admin", "profile": "support_admin"},
        ],
        "employees": [
            {"role_ref": "cashier", "name": "Demo Cashier", "pin_name": "cashier_pin", "pin": "1111"},
            {"role_ref": "manager", "name": "Demo Manager", "pin_name": "manager_pin", "pin": "2222"},
            {"role_ref": "waiter", "name": "Demo Waiter", "pin_name": "waiter_pin", "pin": "3333"},
            {"role_ref": "senior_cashier", "name": "Demo Senior Cashier", "pin_name": "senior_cashier_pin", "pin": "4444"},
            {"role_ref": "kitchen", "name": "Demo Kitchen", "pin_name": "kitchen_pin", "pin": "5555"},
            {"role_ref": "support", "name": "Demo Support", "pin_name": "support_pin", "pin": "9999"},
        ],
        "catalog_folders": [
            {"ref": "bar", "name": "Bar", "sort_order": 10, "parameters": [{"key": "station", "value_type": "string", "value": "bar"}]},
            {"ref": "kitchen", "name": "Kitchen", "sort_order": 20, "parameters": [{"key": "station", "value_type": "string", "value": "hot"}]},
            {"ref": "services", "name": "Services", "sort_order": 30, "parameters": [{"key": "station", "value_type": "string", "value": "service"}]},
            {"ref": "inventory", "name": "Inventory", "sort_order": 40, "parameters": [{"key": "stock_area", "value_type": "string", "value": "main"}]},
        ],
        "catalog_tags": [
            {"ref": "coffee", "name": "Coffee"},
            {"ref": "hot_kitchen", "name": "Hot Kitchen"},
            {"ref": "service", "name": "Service"},
            {"ref": "stop_test", "name": "Stop-list Example"},
        ],
        "catalog_items": [
            {"ref": "espresso", "kind": "dish", "folder_ref": "bar", "name": "Espresso", "base_unit": "portion", "price_minor": 12900, "tags": ["coffee"], "category_ref": "drinks", "station": "bar"},
            {"ref": "cappuccino", "kind": "dish", "folder_ref": "bar", "name": "Cappuccino", "base_unit": "portion", "price_minor": 17900, "tags": ["coffee"], "category_ref": "drinks", "station": "bar"},
            {"ref": "soup", "kind": "dish", "folder_ref": "kitchen", "name": "Tom Yum Soup", "base_unit": "portion", "price_minor": 34900, "tags": ["hot_kitchen"], "category_ref": "kitchen", "station": "hot"},
            {"ref": "sirloin", "kind": "good", "folder_ref": "inventory", "name": "Beef Sirloin", "base_unit": "g", "tags": ["hot_kitchen"]},
            {"ref": "sauce", "kind": "semi_finished", "folder_ref": "inventory", "name": "House Sauce", "base_unit": "g", "tags": ["hot_kitchen"]},
            {"ref": "service_fee", "kind": "service", "folder_ref": "services", "name": "Service Fee", "base_unit": "service", "price_minor": 5000, "tags": ["service"], "category_ref": "services", "station": "service"},
            {"ref": "sold_out_dessert", "kind": "dish", "folder_ref": "kitchen", "name": "Sold Out Cheesecake", "base_unit": "portion", "price_minor": 29900, "tags": ["stop_test"], "category_ref": "kitchen", "station": "cold"},
        ],
        "menu_categories": [
            {"ref": "drinks", "name": "Drinks", "sort_order": 10},
            {"ref": "kitchen", "name": "Kitchen", "sort_order": 20},
            {"ref": "services", "name": "Services", "sort_order": 30},
        ],
        "modifier_groups": [
            {
                "ref": "milk_options",
                "name": "Milk Options",
                "required": False,
                "min_count": 0,
                "max_count": 1,
                "options": [{"name": "Oat Milk", "price_minor": 2500}, {"name": "Lactose Free", "price_minor": 2000}],
                "bindings": [{"target_type": "tag", "target_ref": "coffee", "sort_order": 1}],
            },
            {
                "ref": "soup_spice",
                "name": "Spice Level",
                "required": True,
                "min_count": 1,
                "max_count": 1,
                "options": [{"name": "Mild", "price_minor": 0}, {"name": "Hot", "price_minor": 0}],
                "bindings": [{"target_type": "menu_item", "target_ref": "soup", "sort_order": 1}],
            },
        ],
        "pricing_policies": [
            {"name": "Lunch Discount", "kind": "discount", "scope": "order", "amount_kind": "percentage", "value_basis_points": 500, "application_index": 10, "manual": False},
            {"name": "Service Surcharge", "kind": "surcharge", "scope": "order", "amount_kind": "fixed", "amount_minor": 3000, "application_index": 20, "manual": False},
            {"name": "Manager Manual Discount", "kind": "discount", "scope": "line", "amount_kind": "fixed", "amount_minor": 1000, "application_index": 30, "manual": True, "requires_permission": "pos.pricing.discount.apply"},
        ],
        "recipes": [
            {"owner_ref": "soup", "component_ref": "sirloin", "quantity": 120, "unit": "g", "loss_percent": 5},
            {"owner_ref": "soup", "component_ref": "sauce", "quantity": 30, "unit": "g", "loss_percent": 0},
        ],
        "stop_list": [
            {"catalog_ref": "sold_out_dessert", "available_quantity": 0, "reason": "Demo sold out item", "active": True},
        ],
        "floor": [
            {"ref": "main", "name": "Main Hall", "tables": [{"name": "T1", "seats": 2}, {"name": "T2", "seats": 4}, {"name": "T3", "seats": 6}]},
            {"ref": "patio", "name": "Patio", "tables": [{"name": "P1", "seats": 2}, {"name": "P2", "seats": 4}]},
        ],
    }


def seed_full_system(
    cloud_client,
    pos_client,
    license_client,
    cloud_base_url="http://localhost:8090",
    client_device_id="seed-dev-system-client",
    suffix="",
    wait_seconds=90,
    interval_seconds=2,
    run_minimal_flow=False,
):
    suffix = suffix or command_suffix()
    health = {
        "cloud": root_health(cloud_client),
        "pos": root_health(pos_client),
        "license": root_health(license_client),
    }
    status = request(pos_client, "GET", f"{API_PREFIX}/system/provisioning-status", expected_status=(200,))
    node_device_id = status.get("node_device_id", "")
    if not node_device_id:
        raise RuntimeError("POS Edge did not return node_device_id")
    if status.get("paired"):
        raise RuntimeError("POS Edge is already paired. Reset local backend data before running the full seed.")

    dataset = build_seed_dataset(suffix)
    restaurant = request(cloud_client, "POST", f"{API_PREFIX}/restaurants", dataset["restaurant"], expected_status=(201,))
    restaurant_id = restaurant["id"]

    role_ids = {}
    for role in dataset["roles"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/roles",
            {
                "restaurant_id": restaurant_id,
                "name": f"{role['name']} {suffix}",
                "permissions_json": permissions_json(role["profile"]),
            },
            expected_status=(201,),
        )
        role_ids[role["ref"]] = created["id"]

    employee_ids = {}
    pins = {}
    for employee in dataset["employees"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/employees",
            {
                "restaurant_id": restaurant_id,
                "role_id": role_ids[employee["role_ref"]],
                "name": employee["name"],
                "pin": employee["pin"],
            },
            expected_status=(201,),
        )
        employee_ids[employee["pin_name"].replace("_pin", "")] = created["id"]
        pins[employee["pin_name"]] = employee["pin"]

    folder_ids = {}
    for folder in dataset["catalog_folders"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/catalog/folders",
            {"restaurant_id": restaurant_id, "name": folder["name"], "sort_order": folder["sort_order"]},
            expected_status=(201,),
        )
        folder_ids[folder["ref"]] = created["id"]
        for parameter in folder.get("parameters", []):
            request(
                cloud_client,
                "POST",
                f"{API_PREFIX}/master-data/catalog/folder-parameters",
                {
                    "restaurant_id": restaurant_id,
                    "folder_id": created["id"],
                    "parameter_key": parameter["key"],
                    "value_type": parameter["value_type"],
                    "value_json": json.dumps(parameter["value"], ensure_ascii=False),
                },
                expected_status=(201,),
            )

    tag_ids = {}
    for tag in dataset["catalog_tags"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/catalog/tags",
            {"restaurant_id": restaurant_id, "name": tag["name"], "code": slug(f"{tag['name']}-{suffix}")},
            expected_status=(201,),
        )
        tag_ids[tag["ref"]] = created["id"]

    catalog_ids = {}
    menu_eligible_refs = []
    for item in dataset["catalog_items"]:
        body = {
            "restaurant_id": restaurant_id,
            "kind": item["kind"],
            "name": item["name"],
            "sku": system_sku(item["kind"], item["name"], suffix),
            "base_unit": item["base_unit"],
            "folder_id": folder_ids.get(item.get("folder_ref", ""), ""),
            "kitchen_type": item.get("station", ""),
            "accounting_category": item["kind"],
        }
        created = request(cloud_client, "POST", f"{API_PREFIX}/master-data/catalog/items", body, expected_status=(201,))
        catalog_ids[item["ref"]] = created["id"]
        for tag_ref in item.get("tags", []):
            request(
                cloud_client,
                "POST",
                f"{API_PREFIX}/master-data/catalog/item-tags",
                {"restaurant_id": restaurant_id, "catalog_item_id": created["id"], "tag_id": tag_ids[tag_ref]},
                expected_status=(201,),
            )
        if "price_minor" in item:
            menu_eligible_refs.append(item["ref"])

    category_ids = {}
    for category in dataset["menu_categories"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/menu/categories",
            {"restaurant_id": restaurant_id, "name": category["name"], "sort_order": category["sort_order"]},
            expected_status=(201,),
        )
        category_ids[category["ref"]] = created["id"]

    menu_ids = {}
    items_by_ref = {item["ref"]: item for item in dataset["catalog_items"]}
    for ref in menu_eligible_refs:
        item = items_by_ref[ref]
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/menu/items",
            {
                "restaurant_id": restaurant_id,
                "catalog_item_id": catalog_ids[ref],
                "category_id": category_ids.get(item.get("category_ref", ""), ""),
                "name": item["name"],
                "price": item["price_minor"],
                "currency": dataset["restaurant"]["currency"],
                "availability_json": "{}",
                "station_routing_key": item.get("station", ""),
            },
            expected_status=(201,),
        )
        menu_ids[ref] = created["id"]

    modifier_group_ids = {}
    modifier_option_ids = []
    modifier_binding_ids = []
    for group in dataset["modifier_groups"]:
        created_group = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/modifiers/groups",
            {
                "restaurant_id": restaurant_id,
                "name": group["name"],
                "required": group["required"],
                "min_count": group["min_count"],
                "max_count": group["max_count"],
            },
            expected_status=(201,),
        )
        modifier_group_ids[group["ref"]] = created_group["id"]
        for option in group["options"]:
            created_option = request(
                cloud_client,
                "POST",
                f"{API_PREFIX}/master-data/modifiers/options",
                {
                    "restaurant_id": restaurant_id,
                    "modifier_group_id": created_group["id"],
                    "name": option["name"],
                    "price_minor": option["price_minor"],
                },
                expected_status=(201,),
            )
            modifier_option_ids.append(created_option["id"])
        for binding in group["bindings"]:
            target_type = binding["target_type"]
            target_ref = binding["target_ref"]
            target_id = {
                "menu_item": menu_ids,
                "catalog_item": catalog_ids,
                "folder": folder_ids,
                "tag": tag_ids,
            }[target_type][target_ref]
            created_binding = request(
                cloud_client,
                "POST",
                f"{API_PREFIX}/master-data/modifiers/bindings",
                {
                    "restaurant_id": restaurant_id,
                    "modifier_group_id": created_group["id"],
                    "target_type": target_type,
                    "target_id": target_id,
                    "sort_order": binding["sort_order"],
                },
                expected_status=(201,),
            )
            modifier_binding_ids.append(created_binding["id"])

    pricing_policy_ids = []
    for policy in dataset["pricing_policies"]:
        body = {
            "restaurant_id": restaurant_id,
            "name": policy["name"],
            "kind": policy["kind"],
            "scope": policy["scope"],
            "amount_kind": policy["amount_kind"],
            "amount_minor": policy.get("amount_minor", 0),
            "value_basis_points": policy.get("value_basis_points", 0),
            "application_index": policy["application_index"],
            "manual": policy["manual"],
            "requires_permission": policy.get("requires_permission", ""),
        }
        created = request(cloud_client, "POST", f"{API_PREFIX}/master-data/pricing/policies", body, expected_status=(201,))
        pricing_policy_ids.append(created["id"])

    recipe_ids = []
    for recipe in dataset["recipes"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/recipes/items",
            {
                "restaurant_id": restaurant_id,
                "recipe_owner_catalog_item_id": catalog_ids[recipe["owner_ref"]],
                "component_catalog_item_id": catalog_ids[recipe["component_ref"]],
                "quantity": recipe["quantity"],
                "unit": recipe["unit"],
                "loss_percent": recipe["loss_percent"],
            },
            expected_status=(201,),
        )
        recipe_ids.append(created["id"])

    stop_list_ids = []
    for entry in dataset["stop_list"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/inventory/stop-list",
            {
                "restaurant_id": restaurant_id,
                "catalog_item_id": catalog_ids[entry["catalog_ref"]],
                "available_quantity": entry["available_quantity"],
                "reason": entry["reason"],
                "active": entry["active"],
            },
            expected_status=(201,),
        )
        stop_list_ids.append(created["id"])

    hall_ids = []
    table_ids = []
    for hall in dataset["floor"]:
        created_hall = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/floor/halls",
            {"restaurant_id": restaurant_id, "name": hall["name"]},
            expected_status=(201,),
        )
        hall_ids.append(created_hall["id"])
        for table in hall["tables"]:
            created_table = request(
                cloud_client,
                "POST",
                f"{API_PREFIX}/master-data/floor/tables",
                {"restaurant_id": restaurant_id, "hall_id": created_hall["id"], "name": table["name"], "seats": table["seats"]},
                expected_status=(201,),
            )
            table_ids.append(created_table["id"])

    publication = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/master-data/publications",
        {"restaurant_id": restaurant_id, "node_device_id": node_device_id, "published_by": "seed-dev-system"},
        expected_status=(201,),
    )

    pairing = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/restaurants/{restaurant_id}/devices/generate-pairing-code",
        {"node_device_id": node_device_id, "display_name": f"POS Terminal {suffix}", "expires_in_minutes": 30},
        expected_status=(201,),
    )
    pairing_code = pairing["pairing_code"]
    paired = request(pos_client, "POST", f"{API_PREFIX}/system/provisioning/pair-via-license", {"pairing_code": pairing_code}, expected_status=(200,))
    verify_pos_ready(pos_client, restaurant_id, node_device_id, client_device_id, pins["manager_pin"], wait_seconds, interval_seconds)

    summary = {
        "restaurant_id": restaurant_id,
        "node_device_id": node_device_id,
        "pairing_code": pairing_code,
        "pairing_status": paired,
        "cloud_base_url": cloud_base_url,
        "generated_at_unix": int(time.time()),
        "suffix": suffix,
        "pins": pins,
        "employee_ids": employee_ids,
        "role_ids": role_ids,
        "hall_ids": hall_ids,
        "table_ids": table_ids,
        "catalog_item_ids": list(catalog_ids.values()),
        "catalog_item_refs": catalog_ids,
        "menu_item_ids": list(menu_ids.values()),
        "menu_item_refs": menu_ids,
        "modifier_group_ids": list(modifier_group_ids.values()),
        "modifier_option_ids": modifier_option_ids,
        "modifier_binding_ids": modifier_binding_ids,
        "pricing_policy_ids": pricing_policy_ids,
        "recipe_item_ids": recipe_ids,
        "stop_list_ids": stop_list_ids,
        "publication_id": publication["id"],
        "health": health,
    }
    if run_minimal_flow:
        summary["minimal_flow"] = run_minimal_flow_smoke(
            cloud_client,
            pos_client,
            restaurant_id=restaurant_id,
            node_device_id=node_device_id,
            client_device_id=client_device_id,
            pins=pins,
            table_ids=table_ids,
            menu_refs=menu_ids,
            catalog_refs=catalog_ids,
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
        )
    return summary


def verify_pos_ready(pos_client, restaurant_id, node_device_id, client_device_id, manager_pin, wait_seconds, interval_seconds):
    login = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/auth/pin-login",
        {"node_device_id": node_device_id, "client_device_id": client_device_id, "pin": manager_pin},
        expected_status=(201, 200),
    )
    session_id = login.get("session", {}).get("id")
    actor_id = login.get("actor", {}).get("employee_id")
    if not session_id:
        raise RuntimeError("POS PIN login did not return session id")
    headers = {
        "X-Node-Device-ID": node_device_id,
        "X-Client-Device-ID": client_device_id,
        "X-Session-ID": session_id,
        "X-Actor-Employee-ID": actor_id or "",
    }
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        halls = request(pos_client, "GET", f"{API_PREFIX}/halls", expected_status=(200,), query={"restaurant_id": restaurant_id}, headers=headers)
        menu = request(pos_client, "GET", f"{API_PREFIX}/menu/items", expected_status=(200,), headers=headers)
        sync_status = request(pos_client, "GET", f"{API_PREFIX}/sync/status", expected_status=(200,), headers=headers)
        if halls and menu:
            return {"halls": len(halls), "menu_items": len(menu), "sync_status": sync_status}
        if time.monotonic() >= deadline:
            raise RuntimeError("POS Edge did not expose seeded halls/menu before timeout")
        time.sleep(max(0, interval_seconds))


def run_minimal_flow_smoke(
    cloud_client,
    pos_client,
    restaurant_id,
    node_device_id,
    client_device_id,
    pins,
    table_ids,
    menu_refs,
    catalog_refs,
    wait_seconds,
    interval_seconds,
):
    waiter_headers = login_pos(pos_client, node_device_id, client_device_id, pins["waiter_pin"])
    ensure_employee_shift(pos_client, restaurant_id, waiter_headers, "minimal-waiter")
    menu_items = request(pos_client, "GET", f"{API_PREFIX}/menu/items", expected_status=(200,), headers=waiter_headers)
    smoke_menu_id = menu_refs.get("soup") or first_menu_item_id(menu_items, excluded_id=menu_refs.get("sold_out_dessert", ""))
    if not smoke_menu_id:
        raise RuntimeError("minimal flow requires at least one sellable menu item")
    smoke_menu_item = find_by_id(menu_items, smoke_menu_id)
    if not smoke_menu_item:
        raise RuntimeError(f"POS Edge menu does not expose smoke menu item {smoke_menu_id}")
    table_id = table_ids[0] if table_ids else ""
    if not table_id:
        raise RuntimeError("minimal flow requires at least one table")

    order = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/orders",
        {
            "command_id": f"cmd-minimal-{command_suffix()}-waiter-order",
            "restaurant_id": restaurant_id,
            "table_id": table_id,
            "table_name": "Minimal Smoke",
            "guest_count": 1,
        },
        expected_status=(201,),
        headers=waiter_headers,
    )

    blocked_sale = {}
    stopped_menu_id = menu_refs.get("sold_out_dessert", "")
    if stopped_menu_id:
        blocked_sale = request(
            pos_client,
            "POST",
            f"{API_PREFIX}/orders/{order['id']}/lines",
            {
                "command_id": f"cmd-minimal-{command_suffix()}-blocked-sale",
                "menu_item_id": stopped_menu_id,
                "quantity": 1,
            },
            expected_status=(409,),
            headers=waiter_headers,
        )

    line = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/orders/{order['id']}/lines",
        {
            "command_id": f"cmd-minimal-{command_suffix()}-waiter-line",
            "menu_item_id": smoke_menu_id,
            "quantity": 1,
            "selected_modifiers": selected_required_modifiers(smoke_menu_item),
        },
        expected_status=(201,),
        headers=waiter_headers,
    )
    precheck = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/orders/{order['id']}/precheck",
        {"command_id": f"cmd-minimal-{command_suffix()}-waiter-precheck"},
        expected_status=(201,),
        headers=waiter_headers,
    )

    cashier_headers = login_pos(pos_client, node_device_id, client_device_id, pins["cashier_pin"])
    ensure_employee_shift(pos_client, restaurant_id, cashier_headers, "minimal-cashier")
    ensure_cash_session(pos_client, restaurant_id, cashier_headers, "minimal-cashier")
    payment = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/prechecks/{precheck['id']}/payments",
        {
            "command_id": f"cmd-minimal-{command_suffix()}-cashier-payment",
            "method": "cash",
            "amount": precheck["total"],
            "currency": precheck.get("currency", "RUB"),
        },
        expected_status=(201,),
        headers=cashier_headers,
    )
    paid_order = request(pos_client, "GET", f"{API_PREFIX}/orders/{order['id']}", expected_status=(200,), headers=cashier_headers)
    check = paid_order.get("check") or {}
    check_id = check.get("id", "")
    if not check_id:
        raise RuntimeError("minimal flow payment did not create final check")

    check_closed = wait_for_cloud_event(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="CheckClosed",
        aggregate_id=check_id,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    ledger = wait_for_inventory_ledger(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_id=check_closed["event_id"],
        order_line_id=line["id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    ledger_catalog_ids = sorted({item.get("catalog_item_id", "") for item in ledger if item.get("catalog_item_id")})
    expected_components = sorted(
        value
        for key, value in catalog_refs.items()
        if key in ("sirloin", "sauce")
    )
    if expected_components and not set(expected_components).issubset(set(ledger_catalog_ids)):
        raise RuntimeError(f"Cloud inventory ledger did not include recipe components: expected {expected_components}, got {ledger_catalog_ids}")

    return {
        "order_id": order["id"],
        "order_line_id": line["id"],
        "precheck_id": precheck["id"],
        "payment_id": payment["id"],
        "check_id": check_id,
        "check_closed_event_id": check_closed["event_id"],
        "ledger_entry_count": len(ledger),
        "ledger_catalog_item_ids": ledger_catalog_ids,
        "blocked_sale_error_code": blocked_sale.get("error_code", ""),
    }


def login_pos(pos_client, node_device_id, client_device_id, pin):
    login = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/auth/pin-login",
        {
            "command_id": f"cmd-minimal-{command_suffix()}-pin-login",
            "node_device_id": node_device_id,
            "client_device_id": client_device_id,
            "pin": pin,
        },
        expected_status=(200, 201),
    )
    session_id = login.get("session", {}).get("id")
    actor_id = login.get("actor", {}).get("employee_id")
    if not session_id or not actor_id:
        raise RuntimeError("POS PIN login did not return session and actor ids")
    return {
        "X-Node-Device-ID": node_device_id,
        "X-Client-Device-ID": client_device_id,
        "X-Session-ID": session_id,
        "X-Actor-Employee-ID": actor_id,
    }


def ensure_employee_shift(pos_client, restaurant_id, headers, prefix):
    response = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/employee-shifts/open",
        {
            "command_id": f"cmd-{prefix}-{command_suffix()}-open-shift",
            "restaurant_id": restaurant_id,
            "opened_by_employee_id": headers["X-Actor-Employee-ID"],
        },
        expected_status=(201, 409),
        headers=headers,
    )
    if response.get("id"):
        return response
    return request(pos_client, "GET", f"{API_PREFIX}/employee-shifts/current", expected_status=(200,), headers=headers)


def ensure_cash_session(pos_client, restaurant_id, headers, prefix):
    response = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/cash-shifts/open",
        {
            "command_id": f"cmd-{prefix}-{command_suffix()}-open-cash",
            "restaurant_id": restaurant_id,
            "opened_by_employee_id": headers["X-Actor-Employee-ID"],
            "opening_cash_amount": 0,
        },
        expected_status=(201, 409),
        headers=headers,
    )
    if response.get("id"):
        return response
    return request(pos_client, "GET", f"{API_PREFIX}/cash-shifts/current", expected_status=(200,), headers=headers)


def selected_required_modifiers(menu_item):
    out = []
    for group in menu_item.get("modifier_groups", []) or []:
        if not group.get("required"):
            continue
        options = group.get("options", []) or []
        if not options:
            continue
        out.append({
            "modifier_group_id": group["id"],
            "modifier_option_id": options[0]["id"],
            "quantity": max(1, int(group.get("min_count") or 1)),
        })
    return out


def find_by_id(items, item_id):
    for item in items:
        if item.get("id") == item_id:
            return item
    return None


def first_menu_item_id(items, excluded_id=""):
    for item in items:
        item_id = item.get("id", "")
        if item_id and item_id != excluded_id:
            return item_id
    return ""


def wait_for_cloud_event(cloud_client, restaurant_id, event_type, aggregate_id, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            cloud_client,
            "GET",
            f"{API_PREFIX}/sync/edge-events",
            expected_status=(200,),
            query={"restaurant_id": restaurant_id, "event_type": event_type, "limit": 50},
        )
        for item in items:
            if item.get("aggregate_id") == aggregate_id:
                return item
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud did not receive {event_type} for aggregate {aggregate_id} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_inventory_ledger(cloud_client, restaurant_id, source_event_id, order_line_id, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            cloud_client,
            "GET",
            f"{API_PREFIX}/inventory/stock-ledger",
            expected_status=(200,),
            query={
                "restaurant_id": restaurant_id,
                "source_event_type": "CheckClosed",
                "source_event_id": source_event_id,
                "order_line_id": order_line_id,
                "limit": 50,
            },
        )
        if items:
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud inventory ledger did not receive CheckClosed ledger rows for order line {order_line_id}")
        time.sleep(max(0, interval_seconds))


def write_summary(path, summary):
    if not path:
        return
    target = pathlib.Path(path)
    target.parent.mkdir(parents=True, exist_ok=True)
    with target.open("w", encoding="utf-8") as fh:
        json.dump(summary, fh, ensure_ascii=False, indent=2)
        fh.write("\n")


def parse_args(argv=None):
    parser = argparse.ArgumentParser(description="Seed complete local development data and pair POS Edge.")
    parser.add_argument("--cloud-base", default="http://localhost:8090")
    parser.add_argument("--pos-base", default="http://localhost:8080")
    parser.add_argument("--license-base", default="http://localhost:8095")
    parser.add_argument("--client-device-id", default="seed-dev-system-client")
    parser.add_argument("--suffix", default="")
    parser.add_argument("--output", default=DEFAULT_OUTPUT)
    parser.add_argument("--wait-seconds", type=int, default=90)
    parser.add_argument("--interval-seconds", type=float, default=2)
    parser.add_argument("--run-minimal-flow", action="store_true", help="Run Cloud publication -> Edge sync -> waiter order -> cashier check -> Cloud inventory ledger smoke.")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(argv)
    summary = seed_full_system(
        JsonClient(args.cloud_base),
        JsonClient(args.pos_base),
        JsonClient(args.license_base),
        cloud_base_url=normalize_base_url(args.cloud_base),
        client_device_id=args.client_device_id,
        suffix=args.suffix,
        wait_seconds=args.wait_seconds,
        interval_seconds=args.interval_seconds,
        run_minimal_flow=args.run_minimal_flow,
    )
    write_summary(args.output, summary)
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
