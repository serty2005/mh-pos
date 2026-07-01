#!/usr/bin/env python3
import argparse
import datetime
import json
import pathlib
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
import http.cookiejar


API_PREFIX = "/api/v1"
SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
DEFAULT_OUTPUT = str(SCRIPT_DIR / ".seed-dev-system-summary.json")
DEFAULT_RECEIPT_TEMPLATE_DIR = SCRIPT_DIR.parent / "shared" / "platform" / "receipt" / "engine" / "templates"

# Карта расширения seed/smoke: новый Cloud-owned справочник или stream
# добавляется здесь вместе с dataset key, publication stream и POS read check.
CLOUD_OWNED_SEED_SURFACES = (
    {"dataset_key": "restaurant", "publication_stream": "restaurants", "pos_read_check": "pin_login"},
    {"dataset_key": "roles", "publication_stream": "staff", "pos_read_check": "pin_login"},
    {"dataset_key": "employees", "publication_stream": "staff", "pos_read_check": "pin_login"},
    {"dataset_key": "floor", "publication_stream": "floor", "pos_read_check": "halls"},
    {"dataset_key": "catalog_items", "publication_stream": "catalog", "pos_read_check": "catalog_items"},
    {"dataset_key": "catalog_folders", "publication_stream": "catalog", "pos_read_check": "catalog_items"},
    {"dataset_key": "catalog_tags", "publication_stream": "catalog", "pos_read_check": "catalog_items"},
    {"dataset_key": "modifier_groups", "publication_stream": "catalog", "pos_read_check": "menu_items"},
    {"dataset_key": "menu_categories", "publication_stream": "menu", "pos_read_check": "menu_items"},
    {"dataset_key": "pricing_policies", "publication_stream": "pricing_policy", "pos_read_check": "menu_items"},
    {"dataset_key": "recipes", "publication_stream": "recipes", "pos_read_check": "kitchen_recipe"},
    {"dataset_key": "stop_list", "publication_stream": "inventory_reference", "pos_read_check": "blocked_sale"},
    {"dataset_key": "receipt_templates", "publication_stream": "receipt_templates", "pos_read_check": "sync_status"},
    {"dataset_key": "printers", "publication_stream": "printers", "pos_read_check": "print_routing"},
)

FORBIDDEN_MUTATING_ROUTE_FRAGMENTS = (
    "/storage/archive/apply",
    "/storage/apply",
    "/archive/apply",
    "/archives/apply",
    "/storage/delete",
    "/storage/reset",
    "/storage/compact",
)

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
    "kitchen": [
        "pos.employee_shift.open",
        "pos.employee_shift.close",
        "pos.employee_shift.view_current",
        "pos.employee_shift.recent",
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
        "pos.kitchen.stop_list.view",
        "pos.kitchen.stop_list.update",
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
        "pos.print.status",
        "pos.print.retry",
        "pos.print_routing.view",
        "pos.print_routing.manage",
        "pos.order.cancel_unconfirmed",
        "pos.sync.view",
        "pos.sync.retry_failed",
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
        self.opener = urllib.request.build_opener(
            urllib.request.ProxyHandler({}),
            urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()),
        )

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
    assert_seed_smoke_request_allowed(method, path)
    if hasattr(client, "request"):
        try:
            return client.request(method, path, body, expected_status=expected_status, headers=headers)
        except TypeError:
            return client.request(method, path, body, expected_status=expected_status)
    raise TypeError("client must expose request(method, path, body, expected_status)")


def assert_seed_smoke_request_allowed(method, path):
    method = method.upper()
    if method not in ("POST", "PUT", "PATCH", "DELETE"):
        return
    lowered = path.lower()
    for fragment in FORBIDDEN_MUTATING_ROUTE_FRAGMENTS:
        if fragment in lowered:
            raise RuntimeError(f"seed smoke must not call destructive storage/archive route: {method} {path}")


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


def read_default_receipt_template(name):
    return (DEFAULT_RECEIPT_TEMPLATE_DIR / name).read_text(encoding="utf-8")


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
            {"ref": "kitchen", "name": "Kitchen", "profile": "kitchen"},
            {"ref": "manager", "name": "Manager", "profile": "manager"},
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
            {"ref": "service_fee", "kind": "service", "folder_ref": "services", "name": "Service Fee", "base_unit": "service", "price_minor": 5000, "tags": ["service"], "category_ref": "services", "station": "service", "qr_confirmation_enabled": True, "validity_mode": "cash_session"},
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
            {"owner_ref": "sauce", "component_ref": "sirloin", "quantity": 10, "unit": "g", "loss_percent": 0},
        ],
        "stop_list": [
            {"catalog_ref": "sold_out_dessert", "available_quantity": 0, "reason": "Demo sold out item", "active": True},
        ],
        "receipt_templates": [
            {
                "document_type": "check_nonfiscal",
                "name": "Default non-fiscal check receipt",
                "description": "System default ReceiptLine Level 1 non-fiscal check template",
                "content": read_default_receipt_template("default_precheck.rl"),
                "level": 1,
                "cpl": 48,
                "printer_class": "generic",
                "is_default": True,
            },
            {
                "document_type": "precheck",
                "name": "Default precheck receipt",
                "description": "System default ReceiptLine Level 1 precheck template",
                "content": read_default_receipt_template("default_precheck.rl"),
                "level": 1,
                "cpl": 48,
                "printer_class": "generic",
                "is_default": True,
            },
            {
                "document_type": "ticket",
                "name": "Default service ticket",
                "description": "System default ReceiptLine Level 1 service ticket template",
                "content": read_default_receipt_template("default_ticket.rl"),
                "level": 1,
                "cpl": 48,
                "printer_class": "generic",
                "is_default": True,
            },
        ],
        "floor": [
            {"ref": "main", "name": "Main Hall", "tables": [{"name": "T1", "seats": 2}, {"name": "T2", "seats": 4}, {"name": "T3", "seats": 6}]},
            {"ref": "patio", "name": "Patio", "tables": [{"name": "P1", "seats": 2}, {"name": "P2", "seats": 4}]},
        ],
        "printers": [
            {
                "name": "Local smoke receipt printer",
                "type": "tcp",
                "address": "127.0.0.1",
                "port": 9,
                "document_types": ["check_nonfiscal", "precheck", "ticket"],
                "codepage": "cp437",
                "paper_cut_type": "partial",
                "cpl": 48,
            },
        ],
    }


def validate_seed_extension_plan(dataset):
    missing = sorted(
        item["dataset_key"]
        for item in CLOUD_OWNED_SEED_SURFACES
        if item["dataset_key"] not in dataset
    )
    if missing:
        raise RuntimeError(f"seed extension plan references missing dataset keys: {missing}")


def wait_for_cloud_license_ready(cloud_client, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    last_error = None
    while True:
        try:
            snapshot = request(cloud_client, "GET", f"{API_PREFIX}/license/entitlements", expected_status=(200,))
            if snapshot.get("status") == "active":
                return snapshot
            last_error = f"unexpected entitlement status: {snapshot}"
        except Exception as exc:  # noqa: BLE001 - smoke script reports the last safe HTTP error.
            last_error = str(exc)
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud license entitlements did not become ready before timeout: {last_error}")
        time.sleep(max(0, interval_seconds))


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
    run_kitchen_process_smoke=False,
    license_admin_token="local-development-only",
):
    suffix = suffix or command_suffix()
    health = {
        "cloud": root_health(cloud_client),
        "pos": root_health(pos_client),
        "license": root_health(license_client),
    }
    entitlement_version = int(time.time())
    issued_at = datetime.datetime.now(datetime.timezone.utc)
    canonical_modules = (
        "cloud-subscription",
        "table-mode",
        "kitchen-space",
        "warehouse-mode",
        "waiter-space",
        "telegram-worker",
        "ticket-mode",
    )
    entitlement_body = {
        "version": entitlement_version,
        "status": "active",
        "entitlements": {module_id: True for module_id in canonical_modules},
        "issued_at": issued_at.isoformat().replace("+00:00", "Z"),
        "expires_at": (issued_at + datetime.timedelta(days=30)).isoformat().replace("+00:00", "Z"),
    }
    request(
        license_client,
        "POST",
        f"{API_PREFIX}/admin/login",
        {"username": "admin", "password": license_admin_token},
        expected_status=(200,),
    )
    for server_id in ("cloud-local", "edge-local"):
        request(
            license_client,
            "PUT",
            f"{API_PREFIX}/entitlements/local-tenant/{server_id}",
            entitlement_body,
            expected_status=(200,),
        )
    wait_for_cloud_license_ready(cloud_client, wait_seconds, interval_seconds)
    status = request(pos_client, "GET", f"{API_PREFIX}/system/provisioning-status", expected_status=(200,))
    node_device_id = status.get("node_device_id", "")
    if not node_device_id:
        raise RuntimeError("POS Edge did not return node_device_id")
    if status.get("paired"):
        raise RuntimeError("POS Edge is already paired. Reset local backend data before running the full seed.")

    dataset = build_seed_dataset(suffix)
    validate_seed_extension_plan(dataset)
    restaurant = request(cloud_client, "POST", f"{API_PREFIX}/restaurants", dataset["restaurant"], expected_status=(201,))
    restaurant_id = restaurant["id"]

    # Cloud идемпотентно провижинит дефолтную hall-секцию + дефолтный стол при создании
    # ресторана (POS-86) — переиспользуем эту секцию для seed-таблиц зала и заводим одну
    # точку продаж (обязательна для открытия кассовой смены).
    restaurant_sections = request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/master-data/restaurant-sections",
        expected_status=(200,),
        query={"restaurant_id": restaurant_id},
    )
    default_section = next((section for section in restaurant_sections if section.get("is_default")), None)
    if not default_section:
        raise RuntimeError(f"restaurant {restaurant_id} has no default hall section provisioned by Cloud")
    default_section_id = default_section["id"]

    sales_point = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/master-data/sales-points",
        {"restaurant_id": restaurant_id, "name": f"Front Desk {suffix}", "analytics_tag": slug(f"front-desk-{suffix}")},
        expected_status=(201,),
    )
    sales_point_id = sales_point["id"]

    role_ids = {}
    for role in dataset["roles"]:
        created = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/roles",
            {
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
                "restaurant_ids": [restaurant_id],
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
        if item.get("qr_confirmation_enabled"):
            body["qr_confirmation_enabled"] = True
            body["validity_mode"] = item.get("validity_mode", "cash_session")
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
                "tag_id": tag_ids.get((item.get("tags") or [""])[0], ""),
                "tax_profile_id": "",
                "name": item["name"],
                "price": item["price_minor"],
                "currency": dataset["restaurant"]["currency"],
                "runtime_status": "available",
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

    recipe_versions = create_and_approve_recipe_versions(cloud_client, restaurant_id, catalog_ids, dataset["recipes"], employee_ids["manager"])

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

    receipt_template_ids = seed_receipt_templates(cloud_client, dataset["receipt_templates"])
    printer_ids = seed_printers(cloud_client, restaurant_id, dataset["printers"])

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
                {
                    "restaurant_id": restaurant_id,
                    "hall_id": created_hall["id"],
                    "section_id": default_section_id,
                    "name": table["name"],
                    "seats": table["seats"],
                },
                expected_status=(201,),
            )
            table_ids.append(created_table["id"])

    pairing = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/restaurants/{restaurant_id}/devices/generate-pairing-code",
        {"display_name": f"POS Terminal {suffix}", "expires_in_minutes": 30},
        expected_status=(201,),
    )
    pairing_code = pairing["pairing_code"]
    paired = request(pos_client, "POST", f"{API_PREFIX}/system/provisioning/pair-via-license", {"pairing_code": pairing_code}, expected_status=(200,))
    verify_pos_ready(pos_client, restaurant_id, node_device_id, client_device_id, pins["manager_pin"], wait_seconds, interval_seconds)
    route_headers = login_pos(pos_client, node_device_id, client_device_id, pins["manager_pin"])
    routing_printers = wait_for_print_routing_ready(pos_client, route_headers, sales_point_id, default_section_id, wait_seconds, interval_seconds)
    print_route_ids = [
        route["id"] for route in seed_print_routes(pos_client, route_headers, routing_printers[0]["id"], sales_point_id, default_section_id)
    ]
    publication = request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/restaurants/{restaurant_id}/master-data/publication-state",
        expected_status=(200,),
    )
    delivery_status = request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/restaurants/{restaurant_id}/master-data/delivery-status",
        expected_status=(200,),
    )

    summary = {
        "restaurant_id": restaurant_id,
        "node_device_id": node_device_id,
        "pairing_code": pairing_code,
        "pairing_id": pairing.get("pairing_id", ""),
        "pairing_status": paired,
        "cloud_base_url": cloud_base_url,
        "generated_at_unix": int(time.time()),
        "suffix": suffix,
        "pins": pins,
        "employee_ids": employee_ids,
        "role_ids": role_ids,
        "hall_ids": hall_ids,
        "table_ids": table_ids,
        "default_section_id": default_section_id,
        "sales_point_id": sales_point_id,
        "catalog_item_ids": list(catalog_ids.values()),
        "catalog_item_refs": catalog_ids,
        "menu_item_ids": list(menu_ids.values()),
        "menu_item_refs": menu_ids,
        "modifier_group_ids": list(modifier_group_ids.values()),
        "modifier_option_ids": modifier_option_ids,
        "modifier_binding_ids": modifier_binding_ids,
        "pricing_policy_ids": pricing_policy_ids,
        "recipe_version_ids": [item["version_id"] for item in recipe_versions],
        "recipe_line_ids": [line_id for item in recipe_versions for line_id in item["line_ids"]],
        "recipe_suggestion_ids": [item["suggestion_id"] for item in recipe_versions],
        "stop_list_ids": stop_list_ids,
        "receipt_template_ids": receipt_template_ids,
        "printer_ids": printer_ids,
        "print_route_ids": print_route_ids,
        "publication_id": publication["id"],
        "delivery_status": delivery_status,
        "health": health,
        "entitlement_version": entitlement_version,
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
            sales_point_id=sales_point_id,
            menu_refs=menu_ids,
            catalog_refs=catalog_ids,
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
        )
    if run_kitchen_process_smoke:
        summary["kitchen_process_smoke"] = run_kitchen_process_smoke_flow(
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


def seed_receipt_templates(cloud_client, templates):
    ids = []
    for template in templates:
        query = {
            "document_type": template["document_type"],
            "is_default": "true",
            "is_active": "true",
        }
        existing = request(cloud_client, "GET", f"{API_PREFIX}/receipt-templates", expected_status=(200,), query=query)
        matched = next(
            (
                item
                for item in existing
                if item.get("document_type") == template["document_type"]
                and item.get("is_default") is True
                and not item.get("restaurant_id", "")
            ),
            None,
        )
        if matched:
            updated = request(
                cloud_client,
                "PUT",
                f"{API_PREFIX}/receipt-templates/{matched['id']}",
                template,
                expected_status=(200,),
            )
            ids.append(updated["id"])
            continue
        created = request(cloud_client, "POST", f"{API_PREFIX}/receipt-templates", template, expected_status=(201,))
        ids.append(created["id"])
    return ids


def seed_printers(cloud_client, restaurant_id, printers):
    ids = []
    for printer in printers:
        body = dict(printer)
        body["restaurant_id"] = restaurant_id
        created = request(cloud_client, "POST", f"{API_PREFIX}/printers", body, expected_status=(201,))
        ids.append(created["id"])
    return ids


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
        receipt_templates_synced = int(sync_status.get("last_cloud_version") or 0) > 0
        if halls and menu and receipt_templates_synced:
            return {"halls": len(halls), "menu_items": len(menu), "sync_status": sync_status}
        if time.monotonic() >= deadline:
            raise RuntimeError("POS Edge did not expose seeded halls/menu and applied Cloud->Edge streams before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_print_routing_ready(pos_client, headers, sales_point_id, section_id, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        printers = request(pos_client, "GET", f"{API_PREFIX}/print-routing/printers", expected_status=(200,), headers=headers)
        sales_points = request(pos_client, "GET", f"{API_PREFIX}/print-routing/sales-points", expected_status=(200,), headers=headers)
        sections = request(pos_client, "GET", f"{API_PREFIX}/print-routing/sections", expected_status=(200,), headers=headers)
        if (
            printers
            and any(item.get("id") == sales_point_id for item in sales_points)
            and any(item.get("id") == section_id for item in sections)
        ):
            return printers
        if time.monotonic() >= deadline:
            raise RuntimeError("POS Edge did not expose synced printers/sales points/sections before timeout")
        time.sleep(max(0, interval_seconds))


def seed_print_routes(pos_client, headers, printer_id, sales_point_id, section_id):
    required = True
    routes = [
        {"document_type": "check_nonfiscal", "scope_type": "sales_point", "scope_id": sales_point_id, "printer_id": printer_id, "is_required": required, "sort_order": 10},
        {"document_type": "precheck", "scope_type": "section", "scope_id": section_id, "printer_id": printer_id, "is_required": required, "sort_order": 20},
        {"document_type": "ticket", "scope_type": "section", "scope_id": section_id, "printer_id": printer_id, "is_required": required, "sort_order": 30},
    ]
    return [
        request(pos_client, "POST", f"{API_PREFIX}/print-routing/routes", route, expected_status=(201,), headers=headers)
        for route in routes
    ]


def create_and_approve_recipe_versions(cloud_client, restaurant_id, catalog_ids, recipes, manager_employee_id):
    by_owner = {}
    for recipe in recipes:
        by_owner.setdefault(recipe["owner_ref"], []).append(recipe)

    created_versions = []
    for owner_ref, lines in by_owner.items():
        draft = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/recipes/versions/drafts",
            {
                "restaurant_id": restaurant_id,
                "owner_catalog_item_id": catalog_ids[owner_ref],
                "name": f"{owner_ref} demo recipe",
                "yield_quantity": 1,
                "yield_unit": "portion",
                "created_by_employee_id": manager_employee_id,
                "submit_for_review": False,
                "reason": "seed manager recipe draft",
                "lines": [
                    {
                        "component_catalog_item_id": catalog_ids[recipe["component_ref"]],
                        "quantity": recipe["quantity"],
                        "unit": recipe["unit"],
                        "loss_percent": recipe["loss_percent"],
                    }
                    for recipe in lines
                ],
            },
            expected_status=(201,),
        )
        version = draft.get("version") or {}
        version_id = version.get("id", "")
        if not version_id:
            raise RuntimeError(f"Cloud recipe draft did not return version id for owner {owner_ref}: {draft}")

        suggestion = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/recipes/versions/{version_id}/submit",
            {
                "submitted_by_employee_id": manager_employee_id,
                "reason": "seed manager recipe review",
            },
            expected_status=(200,),
        )
        suggestion_id = suggestion.get("id", "")
        if not suggestion_id:
            raise RuntimeError(f"Cloud recipe submit did not return suggestion id for version {version_id}: {suggestion}")

        approved = request(
            cloud_client,
            "POST",
            f"{API_PREFIX}/master-data/recipe-suggestions/{suggestion_id}/approve",
            {
                "reviewed_by_employee_id": manager_employee_id,
                "review_comment": "approved by seed-dev-system manager flow",
            },
            expected_status=(200,),
        )
        if approved.get("status") != "approved":
            raise RuntimeError(f"Cloud recipe suggestion was not approved for version {version_id}: {approved}")
        created_versions.append({
            "owner_ref": owner_ref,
            "version_id": version_id,
            "line_ids": [line.get("id", "") for line in draft.get("lines", []) if line.get("id", "")],
            "suggestion_id": suggestion_id,
        })
    return created_versions


def run_minimal_flow_smoke(
    cloud_client,
    pos_client,
    restaurant_id,
    node_device_id,
    client_device_id,
    pins,
    table_ids,
    sales_point_id,
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
        if error_code(blocked_sale) != "SALE_STOP_LIST_CONFLICT":
            raise RuntimeError(f"blocked sale returned unexpected error contract: {blocked_sale}")

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
    kitchen_pin = (
        pins.get("kitchen_pin")
        or pins.get("manager_pin")
        or pins.get("senior_cashier_pin")
        or pins.get("cashier_pin")
    )
    if not kitchen_pin:
        raise RuntimeError("minimal flow requires kitchen_pin or manager/senior_cashier/cashier pin for kitchen actions")
    kitchen_headers = login_pos(pos_client, node_device_id, client_device_id, kitchen_pin)
    kitchen_ticket = mark_kitchen_ticket_served(pos_client, line["id"], kitchen_headers)
    item_served = wait_for_cloud_event(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="ItemServed",
        aggregate_id=kitchen_ticket["id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    served_ledger = wait_for_inventory_ledger(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="ItemServed",
        source_event_id=item_served["event_id"],
        order_line_id=line["id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    ticket_line = {}
    ticket_menu_id = menu_refs.get("service_fee", "")
    if ticket_menu_id:
        ticket_menu_item = find_by_id(menu_items, ticket_menu_id)
        if not ticket_menu_item:
            raise RuntimeError(f"POS Edge menu does not expose service ticket menu item {ticket_menu_id}")
        ticket_line = request(
            pos_client,
            "POST",
            f"{API_PREFIX}/orders/{order['id']}/lines",
            {
                "command_id": f"cmd-minimal-{command_suffix()}-ticket-line",
                "menu_item_id": ticket_menu_id,
                "quantity": 1,
                "selected_modifiers": selected_required_modifiers(ticket_menu_item),
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
    ensure_cash_session(pos_client, restaurant_id, cashier_headers, "minimal-cashier", sales_point_id)
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
    issued_tickets = request(pos_client, "GET", f"{API_PREFIX}/checks/{check_id}/tickets", expected_status=(200,), headers=cashier_headers)
    ticket_ids = [item.get("id", "") for item in issued_tickets if item.get("id")]
    if ticket_line and not ticket_ids:
        raise RuntimeError(f"service ticket line {ticket_line.get('id', '')} did not issue ticket units for check {check_id}")
    manager_headers = login_pos(pos_client, node_device_id, client_device_id, pins["manager_pin"])
    print_jobs = wait_for_print_jobs_for_sources(
        pos_client,
        manager_headers,
        [("precheck", precheck["id"])] + [("ticket", ticket_id) for ticket_id in ticket_ids],
        wait_seconds,
        interval_seconds,
    )

    check_closed = wait_for_cloud_event(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="CheckClosed",
        aggregate_id=check_id,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    check_closed_delta = list_inventory_ledger(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="CheckClosed",
        source_event_id=check_closed["event_id"],
        order_line_id=line["id"],
    )
    if check_closed_delta:
        raise RuntimeError(f"CheckClosed created duplicate inventory ledger rows after ItemServed for order line {line['id']}")
    ledger_catalog_ids = sorted({item.get("catalog_item_id", "") for item in served_ledger if item.get("catalog_item_id")})
    expected_components = sorted(
        value
        for key, value in catalog_refs.items()
        if key in ("sirloin", "sauce")
    )
    if expected_components and not set(expected_components).issubset(set(ledger_catalog_ids)):
        raise RuntimeError(f"Cloud inventory ledger did not include recipe components: expected {expected_components}, got {ledger_catalog_ids}")
    stock_balances = wait_for_inventory_stock_balances(
        cloud_client,
        restaurant_id=restaurant_id,
        expected_catalog_item_ids=ledger_catalog_ids,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )

    served_olap_events = wait_for_olap_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="ItemServed",
        event_ids={item_served["event_id"]},
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    check_closed_olap_events = wait_for_olap_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="CheckClosed",
        event_ids={check_closed["event_id"]},
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    served_olap_stock_moves = wait_for_olap_stock_moves(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="ItemServed",
        source_event_id=item_served["event_id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    stock_move_summary = wait_for_olap_stock_move_summary(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="ItemServed",
        expected_catalog_item_ids=ledger_catalog_ids,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    sales_kitchen_summary = wait_for_sales_kitchen_summary(
        cloud_client,
        restaurant_id=restaurant_id,
        group_by="event_type",
        expected_group_keys={"ItemServed", "CheckClosed"},
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    kitchen_timing_summary = wait_for_kitchen_timing_summary(
        cloud_client,
        restaurant_id=restaurant_id,
        group_by="business_date",
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )

    return {
        "order_id": order["id"],
        "order_line_id": line["id"],
        "ticket_order_line_id": ticket_line.get("id", ""),
        "precheck_id": precheck["id"],
        "payment_id": payment["id"],
        "check_id": check_id,
        "issued_ticket_ids": ticket_ids,
        "print_job_ids": sorted(job["id"] for job in print_jobs),
        "print_job_statuses": sorted({job.get("status", "") for job in print_jobs}),
        "kitchen_ticket_id": kitchen_ticket["id"],
        "item_served_event_id": item_served["event_id"],
        "check_closed_event_id": check_closed["event_id"],
        "served_ledger_entry_count": len(served_ledger),
        "check_closed_delta_entry_count": len(check_closed_delta),
        "ledger_catalog_item_ids": ledger_catalog_ids,
        "stock_balance_count": len(stock_balances),
        "stock_balance_catalog_item_ids": sorted({item.get("catalog_item_id", "") for item in stock_balances if item.get("catalog_item_id", "")}),
        "olap_item_served_event_count": len(served_olap_events),
        "olap_check_closed_event_count": len(check_closed_olap_events),
        "olap_item_served_stock_move_count": len(served_olap_stock_moves),
        "olap_stock_move_summary_count": len(stock_move_summary),
        "olap_sales_kitchen_summary_group_keys": sorted({item.get("group_key", "") for item in sales_kitchen_summary if item.get("group_key", "")}),
        "olap_kitchen_timing_summary_count": len(kitchen_timing_summary),
        "blocked_sale_error_code": error_code(blocked_sale),
    }


def wait_for_print_jobs_for_sources(pos_client, headers, expected_sources, wait_seconds, interval_seconds):
    expected = {(document_type, source_id) for document_type, source_id in expected_sources if source_id}
    if not expected:
        raise RuntimeError("minimal flow print smoke requires at least one expected print source")
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        jobs = request(pos_client, "GET", f"{API_PREFIX}/print/jobs", expected_status=(200,), query={"limit": 100}, headers=headers)
        matched = [
            job for job in jobs
            if (job.get("document_type", ""), job.get("source_id", "")) in expected
        ]
        found = {(job.get("document_type", ""), job.get("source_id", "")) for job in matched}
        if expected.issubset(found) and all(job.get("status") in ("succeeded", "failed") for job in matched):
            return matched
        if time.monotonic() >= deadline:
            raise RuntimeError(f"POS Edge print jobs did not reach terminal status before timeout; missing={sorted(expected - found)} matched={matched}")
        time.sleep(max(0, interval_seconds))


def run_kitchen_process_smoke_flow(
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
    kitchen_headers = login_pos(pos_client, node_device_id, client_device_id, pins["kitchen_pin"])
    ensure_employee_shift(pos_client, restaurant_id, waiter_headers, "kitchen-smoke-waiter")
    ensure_employee_shift(pos_client, restaurant_id, kitchen_headers, "kitchen-smoke-kitchen")

    menu_items = request(pos_client, "GET", f"{API_PREFIX}/menu/items", expected_status=(200,), headers=waiter_headers)
    catalog_items = request(pos_client, "GET", f"{API_PREFIX}/catalog/items", expected_status=(200,), headers=kitchen_headers)
    smoke_menu_id = menu_refs.get("soup") or first_menu_item_id(menu_items, excluded_id=menu_refs.get("sold_out_dessert", ""))
    smoke_menu_item = find_by_id(menu_items, smoke_menu_id)
    if not smoke_menu_id or not smoke_menu_item:
        raise RuntimeError("kitchen process smoke requires seeded soup menu item")
    soup_catalog_id = catalog_refs.get("soup") or smoke_menu_item.get("catalog_item_id", "")
    sirloin_catalog_id = catalog_refs.get("sirloin", "")
    sauce_catalog_id = catalog_refs.get("sauce", "")
    if not soup_catalog_id or not sirloin_catalog_id or not sauce_catalog_id:
        raise RuntimeError("kitchen process smoke requires soup, sirloin and sauce catalog ids")
    recipe = request(pos_client, "GET", f"{API_PREFIX}/kitchen/catalog/items/{soup_catalog_id}/recipe", expected_status=(200,), headers=kitchen_headers)
    if not recipe.get("recipe_version", {}).get("id") or not recipe.get("ingredients"):
        raise RuntimeError(f"POS Edge did not expose synced recipe for soup: {recipe}")
    if not table_ids:
        raise RuntimeError("kitchen process smoke requires at least one table")

    order = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/orders",
        {
            "command_id": f"cmd-kitchen-smoke-{command_suffix()}-waiter-order",
            "restaurant_id": restaurant_id,
            "table_id": table_ids[0],
            "table_name": "Kitchen Smoke",
            "guest_count": 1,
        },
        expected_status=(201,),
        headers=waiter_headers,
    )
    line = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/orders/{order['id']}/lines",
        {
            "command_id": f"cmd-kitchen-smoke-{command_suffix()}-waiter-line",
            "menu_item_id": smoke_menu_id,
            "quantity": 1,
            "selected_modifiers": selected_required_modifiers(smoke_menu_item),
        },
        expected_status=(201,),
        headers=waiter_headers,
    )
    queue = wait_for_kitchen_order_tile(pos_client, line["id"], kitchen_headers, wait_seconds, interval_seconds)
    ticket = queue["ticket"]
    first_served_ticket = run_kitchen_actions(pos_client, ticket, ("accept", "start", "ready", "serve"), kitchen_headers, "kitchen-smoke-first")
    recalled_ticket = run_kitchen_actions(pos_client, first_served_ticket, ("recall", "start", "ready", "serve"), kitchen_headers, "kitchen-smoke-recall")
    if recalled_ticket.get("status") != "served":
        raise RuntimeError(f"KDS recall/serve-again did not finish in served status: {recalled_ticket}")

    served_events = wait_for_cloud_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="ItemServed",
        aggregate_id=ticket["id"],
        min_count=2,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    status_events = wait_for_cloud_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="KitchenTicketStatusChanged",
        aggregate_id=ticket["id"],
        min_count=8,
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    latest_served_event = served_events[0]
    latest_served_ledger = wait_for_inventory_ledger(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="ItemServed",
        source_event_id=latest_served_event["event_id"],
        order_line_id=line["id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    latest_served_olap_stock_moves = wait_for_olap_stock_moves(
        cloud_client,
        restaurant_id=restaurant_id,
        source_event_type="ItemServed",
        source_event_id=latest_served_event["event_id"],
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    olap_item_served = wait_for_olap_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="ItemServed",
        event_ids={item["event_id"] for item in served_events[:2]},
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )
    olap_status = wait_for_olap_events(
        cloud_client,
        restaurant_id=restaurant_id,
        event_type="KitchenTicketStatusChanged",
        event_ids={item["event_id"] for item in status_events[:2]},
        wait_seconds=wait_seconds,
        interval_seconds=interval_seconds,
    )

    today = time.strftime("%Y-%m-%d", time.gmtime())
    now_iso = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    stock_commands = [
        (
            "receipt",
            "StockReceiptCaptured",
            f"{API_PREFIX}/kitchen/stock-receipts",
            {
                "command_id": f"cmd-kitchen-smoke-{command_suffix()}-receipt",
                "receipt_id": "receipt-1",
                "warehouse_id": "warehouse-main",
                "supplier_counterparty_id": "supplier-demo",
                "supplier_name_snapshot": "Demo Supplier",
                "document_number": f"SMOKE-{command_suffix()}",
                "document_date": today,
                "received_at": now_iso,
                "business_date_local": today,
                "currency": "RUB",
                "items": [{
                    "line_id": f"receipt-line-{command_suffix()}",
                    "catalog_item_id": sirloin_catalog_id,
                    "name_snapshot": "Beef Sirloin",
                    "quantity": "10.000",
                    "unit_code": "g",
                    "unit_cost_minor": 10,
                    "line_total_minor": 100,
                    "currency": "RUB",
                }],
            },
        ),
        (
            "count",
            "InventoryCountCaptured",
            f"{API_PREFIX}/kitchen/inventory-counts",
            {
                "command_id": f"cmd-kitchen-smoke-{command_suffix()}-count",
                "count_id": "count-1",
                "warehouse_id": "warehouse-main",
                "counted_at": now_iso,
                "business_date_local": today,
                "items": [{
                    "line_id": f"count-line-{command_suffix()}",
                    "catalog_item_id": sirloin_catalog_id,
                    "counted_quantity": "7.500",
                    "unit_code": "g",
                }],
            },
        ),
        (
            "write_off",
            "StockWriteOffCaptured",
            f"{API_PREFIX}/kitchen/stock-write-offs",
            {
                "command_id": f"cmd-kitchen-smoke-{command_suffix()}-writeoff",
                "write_off_id": "writeoff-1",
                "warehouse_id": "warehouse-main",
                "written_off_at": now_iso,
                "business_date_local": today,
                "reason_code": "spoilage",
                "reason": "smoke spoilage",
                "items": [{
                    "line_id": f"writeoff-line-{command_suffix()}",
                    "catalog_item_id": sirloin_catalog_id,
                    "quantity": "1.000",
                    "unit_code": "g",
                }],
            },
        ),
        (
            "production",
            "ProductionCompleted",
            f"{API_PREFIX}/kitchen/productions",
            {
                "command_id": f"cmd-kitchen-smoke-{command_suffix()}-production",
                "production_id": "production-1",
                "warehouse_id": "warehouse-main",
                "semi_finished_catalog_item_id": sauce_catalog_id,
                "quantity": "2.000",
                "unit_code": "g",
                "completed_at": now_iso,
                "business_date_local": today,
            },
        ),
    ]
    stock_results = {}
    for name, event_type, path, body in stock_commands:
        captured = request(pos_client, "POST", path, body, expected_status=(201,), headers=kitchen_headers)
        cloud_event = wait_for_cloud_event(
            cloud_client,
            restaurant_id=restaurant_id,
            event_type=event_type,
            aggregate_id=captured["id"],
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
        )
        ledger = wait_for_inventory_ledger(
            cloud_client,
            restaurant_id=restaurant_id,
            source_event_type=event_type,
            source_event_id=cloud_event["event_id"],
            order_line_id="",
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
        )
        olap_stock_moves = wait_for_olap_stock_moves(
            cloud_client,
            restaurant_id=restaurant_id,
            source_event_type=event_type,
            source_event_id=cloud_event["event_id"],
            wait_seconds=wait_seconds,
            interval_seconds=interval_seconds,
        )
        stock_results[name] = {
            "id": captured["id"],
            "warehouse_id": captured.get("warehouse_id", ""),
            "event_type": event_type,
            "cloud_event_id": cloud_event["event_id"],
            "ledger_entry_count": len(ledger),
            "olap_stock_move_count": len(olap_stock_moves),
        }
    stock_balances = list_inventory_stock_balances(
        cloud_client,
        restaurant_id=restaurant_id,
        warehouse_id="warehouse-main",
        business_date_to=today,
    )
    raw_stock_balances = json.dumps(stock_balances, ensure_ascii=False)
    if "payload" in raw_stock_balances or "raw_payload" in raw_stock_balances:
        raise RuntimeError("Cloud stock balances response exposed raw payload")
    balance_item_ids = {item.get("catalog_item_id") for item in stock_balances}
    if sirloin_catalog_id not in balance_item_ids:
        raise RuntimeError("Cloud stock balances did not expose kitchen stock receipt/count/write-off item")

    proposal_group_id = f"proposal-group-{command_suffix()}"
    catalog_suggestion = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/kitchen/catalog-suggestions",
        {
            "command_id": f"cmd-kitchen-smoke-{command_suffix()}-catalog-suggestion",
            "suggestion_id": f"catalog-suggestion-{command_suffix()}",
            "proposal_group_id": proposal_group_id,
            "action": "create",
            "kind": "good",
            "name": f"Smoke Herb {command_suffix()}",
            "sku": system_sku("good", "Smoke Herb", command_suffix()),
            "base_unit": "g",
            "kitchen_type": "hot",
            "accounting_category": "good",
            "reason": "smoke catalog proposal",
        },
        expected_status=(201,),
        headers=kitchen_headers,
    )
    cloud_catalog_suggestion = wait_for_cloud_suggestion(
        cloud_client, "catalog", restaurant_id, catalog_suggestion["id"], wait_seconds, interval_seconds
    )
    review_body = {
        "reviewed_by_employee_id": "seed-dev-system-manager",
        "review_comment": "approved by kitchen process smoke",
    }
    approved_catalog = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/master-data/catalog-suggestions/{cloud_catalog_suggestion['id']}/approve",
        review_body,
        expected_status=(200,),
    )
    approved_catalog_item_id = approved_catalog.get("applied_catalog_item_id", "")
    if not approved_catalog_item_id:
        raise RuntimeError(f"Cloud catalog approval did not return applied catalog item id: {approved_catalog}")
    wait_for_edge_proposal_status(pos_client, "catalog", catalog_suggestion["id"], "approved", kitchen_headers, wait_seconds, interval_seconds)
    updated_catalog_items = wait_for_catalog_item_count(pos_client, len(catalog_items) + 1, kitchen_headers, wait_seconds, interval_seconds)
    if not any(item.get("id") == approved_catalog_item_id for item in updated_catalog_items):
        raise RuntimeError(f"POS Edge did not sync approved catalog item {approved_catalog_item_id}")

    recipe_suggestion = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/kitchen/recipe-suggestions",
        {
            "command_id": f"cmd-kitchen-smoke-{command_suffix()}-recipe-suggestion",
            "suggestion_id": f"recipe-suggestion-{command_suffix()}",
            "proposal_group_id": proposal_group_id,
            "recipe_version_id": recipe["recipe_version"]["id"],
            "owner_catalog_item_id": soup_catalog_id,
            "action": "update_recipe",
            "prep_time_delta_minutes": 1,
            "reason": "smoke recipe proposal",
            "changes": [{
                "action": "add_ingredient",
                "to_catalog_item_id": approved_catalog_item_id,
                "quantity": "1",
                "unit_code": "g",
                "loss_percent": "0",
            }],
        },
        expected_status=(201,),
        headers=kitchen_headers,
    )
    cloud_recipe_suggestion = wait_for_cloud_suggestion(
        cloud_client, "recipe", restaurant_id, recipe_suggestion["id"], wait_seconds, interval_seconds
    )
    approved_recipe = request(
        cloud_client,
        "POST",
        f"{API_PREFIX}/master-data/recipe-suggestions/{cloud_recipe_suggestion['id']}/approve",
        review_body,
        expected_status=(200,),
    )
    wait_for_edge_proposal_status(pos_client, "recipe", recipe_suggestion["id"], "approved", kitchen_headers, wait_seconds, interval_seconds)
    updated_recipe = request(pos_client, "GET", f"{API_PREFIX}/kitchen/catalog/items/{soup_catalog_id}/recipe", expected_status=(200,), headers=kitchen_headers)

    return {
        "edge_catalog_item_count_before": len(catalog_items),
        "edge_catalog_item_count_after": len(updated_catalog_items),
        "recipe_line_count_before": len(recipe.get("ingredients", [])),
        "recipe_line_count_after": len(updated_recipe.get("ingredients", [])),
        "order_id": order["id"],
        "order_line_id": line["id"],
        "kitchen_order_status": queue["order"].get("kitchen_order_status", ""),
        "kitchen_ticket_id": ticket["id"],
        "item_served_event_ids": [item["event_id"] for item in served_events[:2]],
        "latest_item_served_event_id": latest_served_event["event_id"],
        "latest_item_served_ledger_entry_count": len(latest_served_ledger),
        "latest_item_served_olap_stock_move_count": len(latest_served_olap_stock_moves),
        "kitchen_status_event_count": len(status_events),
        "olap_item_served_event_count": len(olap_item_served),
        "olap_status_event_count": len(olap_status),
        "stock": stock_results,
        "stock_balance_count": len(stock_balances),
        "catalog_suggestion_id": catalog_suggestion["id"],
        "recipe_suggestion_id": recipe_suggestion["id"],
        "cloud_catalog_suggestion_id": cloud_catalog_suggestion["id"],
        "cloud_recipe_suggestion_id": cloud_recipe_suggestion["id"],
        "approved_catalog_status": approved_catalog.get("status", ""),
        "approved_recipe_status": approved_recipe.get("status", ""),
    }


def mark_kitchen_ticket_served(pos_client, order_line_id, headers):
    tickets = request(
        pos_client,
        "GET",
        f"{API_PREFIX}/kitchen/tickets",
        expected_status=(200,),
        query={"limit": 100, "offset": 0},
        headers=headers,
    )
    ticket = next((item for item in tickets if item.get("order_line_id") == order_line_id), None)
    if not ticket:
        raise RuntimeError(f"KDS ticket not found for order line {order_line_id}")
    for action in ("accept", "start", "ready", "serve"):
        ticket = request(
            pos_client,
            "POST",
            f"{API_PREFIX}/kitchen/tickets/{ticket['id']}/{action}",
            {"command_id": f"cmd-minimal-{command_suffix()}-kds-{action}"},
            expected_status=(200,),
            headers=headers,
        )
    if ticket.get("status") != "served":
        raise RuntimeError(f"KDS ticket did not reach served status: {ticket}")
    return ticket


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


def ensure_cash_session(pos_client, restaurant_id, headers, prefix, sales_point_id):
    response = request(
        pos_client,
        "POST",
        f"{API_PREFIX}/cash-shifts/open",
        {
            "command_id": f"cmd-{prefix}-{command_suffix()}-open-cash",
            "restaurant_id": restaurant_id,
            "sales_point_id": sales_point_id,
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


def error_code(response):
    if not isinstance(response, dict):
        return ""
    if response.get("error_code"):
        return response.get("error_code", "")
    error = response.get("error")
    if isinstance(error, dict):
        return error.get("code", "") or error.get("error_code", "")
    return response.get("code", "")


def wait_for_kitchen_order_tile(pos_client, order_line_id, headers, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        queue = request(
            pos_client,
            "GET",
            f"{API_PREFIX}/kitchen/order-queue",
            expected_status=(200,),
            query={"limit": 100, "offset": 0},
            headers=headers,
        )
        for order in queue.get("orders", []):
            for ticket in order.get("tickets", []):
                if ticket.get("order_line_id") == order_line_id:
                    return {"order": order, "ticket": ticket}
        if time.monotonic() >= deadline:
            raise RuntimeError(f"kitchen order tile not found for order line {order_line_id}")
        time.sleep(max(0, interval_seconds))


def run_kitchen_actions(pos_client, ticket, actions, headers, prefix):
    current = dict(ticket)
    for action in actions:
        current = request(
            pos_client,
            "POST",
            f"{API_PREFIX}/kitchen/tickets/{current['id']}/{action}",
            {"command_id": f"cmd-{prefix}-{command_suffix()}-{action}"},
            expected_status=(200,),
            headers=headers,
        )
    return current


def wait_for_cloud_event(cloud_client, restaurant_id, event_type, aggregate_id, wait_seconds, interval_seconds):
    return wait_for_cloud_events(cloud_client, restaurant_id, event_type, aggregate_id, 1, wait_seconds, interval_seconds)[0]


def wait_for_cloud_events(cloud_client, restaurant_id, event_type, aggregate_id, min_count, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            cloud_client,
            "GET",
            f"{API_PREFIX}/sync/edge-events",
            expected_status=(200,),
            query={"restaurant_id": restaurant_id, "event_type": event_type, "limit": 50},
        )
        matched = [item for item in items if item.get("aggregate_id") == aggregate_id]
        if len(matched) >= min_count:
            return matched
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud did not receive {min_count} {event_type} events for aggregate {aggregate_id} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_olap_events(cloud_client, restaurant_id, event_type, event_ids, wait_seconds, interval_seconds):
    wanted = {item for item in event_ids if item}
    if not wanted:
        raise RuntimeError(f"OLAP wait requires event ids for {event_type}")
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            cloud_client,
            "GET",
            f"{API_PREFIX}/olap/raw-business-events",
            expected_status=(200,),
            query={"restaurant_id": restaurant_id, "event_type": event_type, "limit": 200},
        )
        matched = [item for item in items if item.get("event_id") in wanted]
        if {item.get("event_id") for item in matched} >= wanted:
            return matched
        if time.monotonic() >= deadline:
            raise RuntimeError(f"ClickHouse raw_business_events did not expose {event_type} ids {sorted(wanted)} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_inventory_ledger(cloud_client, restaurant_id, source_event_type, source_event_id, order_line_id, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_inventory_ledger(
            cloud_client,
            restaurant_id=restaurant_id,
            source_event_type=source_event_type,
            source_event_id=source_event_id,
            order_line_id=order_line_id,
        )
        if items:
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud inventory ledger did not receive {source_event_type} ledger rows for order line {order_line_id}")
        time.sleep(max(0, interval_seconds))


def wait_for_olap_stock_moves(cloud_client, restaurant_id, source_event_type, source_event_id, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_olap_stock_moves(
            cloud_client,
            restaurant_id=restaurant_id,
            source_event_type=source_event_type,
        )
        matched = [item for item in items if item.get("source_event_id") == source_event_id]
        if matched:
            raw = json.dumps(matched, ensure_ascii=False)
            if "payload" in raw:
                raise RuntimeError("OLAP stock moves response exposed raw payload")
            return matched
        if time.monotonic() >= deadline:
            raise RuntimeError(f"ClickHouse olap_stock_moves did not expose {source_event_type} move for event {source_event_id}")
        time.sleep(max(0, interval_seconds))


def wait_for_inventory_stock_balances(cloud_client, restaurant_id, expected_catalog_item_ids, wait_seconds, interval_seconds):
    expected = {item for item in expected_catalog_item_ids if item}
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_inventory_stock_balances(cloud_client, restaurant_id=restaurant_id)
        raw = json.dumps(items, ensure_ascii=False)
        if "payload" in raw or "raw_payload" in raw or "sync_envelope" in raw:
            raise RuntimeError("Cloud stock balances response exposed raw payload")
        if any("quantity_on_hand" not in item or "costing_status" not in item for item in items):
            raise RuntimeError("Cloud stock balances response did not expose materialized balance/costing fields")
        found = {item.get("catalog_item_id") for item in items}
        if expected.issubset(found):
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud materialized stock balances did not expose catalog items {sorted(expected)} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_olap_stock_move_summary(cloud_client, restaurant_id, source_event_type, expected_catalog_item_ids, wait_seconds, interval_seconds):
    expected = {item for item in expected_catalog_item_ids if item}
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_olap_stock_move_summary(
            cloud_client,
            restaurant_id=restaurant_id,
            source_event_type=source_event_type,
            group_by="catalog_item",
        )
        raw = json.dumps(items, ensure_ascii=False)
        if "payload" in raw or "raw_payload" in raw:
            raise RuntimeError("OLAP stock move summary response exposed raw payload")
        found = {item.get("catalog_item_id") or item.get("group_key") for item in items}
        if items and (not expected or expected.issubset(found)):
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"ClickHouse stock-move-summary did not expose {source_event_type} catalog items {sorted(expected)} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_sales_kitchen_summary(cloud_client, restaurant_id, group_by, expected_group_keys, wait_seconds, interval_seconds):
    expected = {item for item in expected_group_keys if item}
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_sales_kitchen_summary(cloud_client, restaurant_id=restaurant_id, group_by=group_by)
        raw = json.dumps(items, ensure_ascii=False)
        if "payload" in raw or "raw_payload" in raw or "margin" in raw.lower() or "cogs" in raw.lower():
            raise RuntimeError("OLAP sales-kitchen summary response exposed raw payload or costing BI fields")
        found = {item.get("group_key", "") for item in items}
        if expected.issubset(found):
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"ClickHouse sales-kitchen-summary did not expose group keys {sorted(expected)} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_kitchen_timing_summary(cloud_client, restaurant_id, group_by, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = list_kitchen_timing_summary(cloud_client, restaurant_id=restaurant_id, group_by=group_by)
        raw = json.dumps(items, ensure_ascii=False)
        if "payload" in raw or "raw_payload" in raw or "margin" in raw.lower() or "cogs" in raw.lower():
            raise RuntimeError("OLAP kitchen timing response exposed raw payload or costing BI fields")
        if any(item.get("ticket_count", 0) > 0 for item in items):
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError("ClickHouse kitchen-timing-summary did not expose KDS timing rows before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_cloud_suggestion(cloud_client, kind, restaurant_id, suggestion_id, wait_seconds, interval_seconds):
    path = {
        "catalog": f"{API_PREFIX}/master-data/catalog-suggestions",
        "recipe": f"{API_PREFIX}/master-data/recipe-suggestions",
    }[kind]
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            cloud_client,
            "GET",
            path,
            expected_status=(200,),
            query={"restaurant_id": restaurant_id, "status": "pending", "limit": 100, "offset": 0},
        )
        for item in items:
            item_suggestion_id = item.get("suggestion_id", "")
            if item_suggestion_id == suggestion_id or item.get("id") == suggestion_id or suggestion_id.startswith(item_suggestion_id + "-"):
                return item
        if time.monotonic() >= deadline:
            raise RuntimeError(f"Cloud did not expose pending {kind} suggestion {suggestion_id} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_edge_proposal_status(pos_client, kind, proposal_id, status, headers, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(
            pos_client,
            "GET",
            f"{API_PREFIX}/kitchen/proposals",
            expected_status=(200,),
            query={"kind": kind, "status": status, "limit": 100, "offset": 0},
            headers=headers,
        ) or []
        for item in items:
            item_id = item.get("id", "")
            if (item_id == proposal_id or proposal_id.startswith(item_id + "-")) and item.get("status") == status:
                return item
        if time.monotonic() >= deadline:
            raise RuntimeError(f"POS Edge proposal {proposal_id} did not reach status {status} before timeout")
        time.sleep(max(0, interval_seconds))


def wait_for_catalog_item_count(pos_client, minimum_count, headers, wait_seconds, interval_seconds):
    deadline = time.monotonic() + max(1, wait_seconds)
    while True:
        items = request(pos_client, "GET", f"{API_PREFIX}/catalog/items", expected_status=(200,), headers=headers)
        if len(items) >= minimum_count:
            return items
        if time.monotonic() >= deadline:
            raise RuntimeError(f"POS Edge catalog item count did not reach {minimum_count} before timeout")
        time.sleep(max(0, interval_seconds))


def list_inventory_ledger(cloud_client, restaurant_id, source_event_type, source_event_id, order_line_id):
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/inventory/stock-ledger",
        expected_status=(200,),
        query={
            "restaurant_id": restaurant_id,
            "source_event_type": source_event_type,
            "source_event_id": source_event_id,
            "order_line_id": order_line_id,
            "limit": 50,
        },
    )


def list_inventory_stock_balances(cloud_client, restaurant_id, warehouse_id="", catalog_item_id="", business_date_to="", costing_status=""):
    query = {
        "restaurant_id": restaurant_id,
        "limit": 50,
    }
    if warehouse_id:
        query["warehouse_id"] = warehouse_id
    if catalog_item_id:
        query["catalog_item_id"] = catalog_item_id
    if business_date_to:
        query["business_date_to"] = business_date_to
    if costing_status:
        query["costing_status"] = costing_status
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/inventory/stock-balances",
        expected_status=(200,),
        query=query,
    )


def list_olap_stock_moves(cloud_client, restaurant_id, source_event_type):
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/olap/stock-moves",
        expected_status=(200,),
        query={
            "restaurant_id": restaurant_id,
            "source_event_type": source_event_type,
            "limit": 50,
        },
    )


def list_olap_stock_move_summary(cloud_client, restaurant_id, source_event_type, group_by):
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/olap/stock-move-summary",
        expected_status=(200,),
        query={
            "restaurant_id": restaurant_id,
            "source_event_type": source_event_type,
            "group_by": group_by,
            "limit": 50,
        },
    )


def list_sales_kitchen_summary(cloud_client, restaurant_id, group_by):
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/olap/sales-kitchen-summary",
        expected_status=(200,),
        query={
            "restaurant_id": restaurant_id,
            "group_by": group_by,
            "limit": 50,
        },
    )


def list_kitchen_timing_summary(cloud_client, restaurant_id, group_by):
    return request(
        cloud_client,
        "GET",
        f"{API_PREFIX}/olap/kitchen-timing-summary",
        expected_status=(200,),
        query={
            "restaurant_id": restaurant_id,
            "group_by": group_by,
            "limit": 50,
        },
    )


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
    parser.add_argument("--run-kitchen-process-smoke", action="store_true", help="Run full kitchen process smoke: KDS recall, ClickHouse trail, stock events, proposals and feedback.")
    parser.add_argument("--license-admin-token", default="local-development-only", help="Provider token for local entitlement bootstrap.")
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
        run_kitchen_process_smoke=args.run_kitchen_process_smoke,
        license_admin_token=args.license_admin_token,
    )
    write_summary(args.output, summary)
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
