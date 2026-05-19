import json
import time
import uuid

from mhpos_http import wait_until


CASHIER_PERMISSIONS = [
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
    "pos.precheck.issue",
    "pos.precheck.view",
    "pos.precheck.reprint",
    "pos.payment.cash",
    "pos.payment.card.manual",
    "pos.check.view",
]

MANAGER_PERMISSIONS = CASHIER_PERMISSIONS + [
    "pos.cash_session.close",
    "pos.cash_drawer.record_event",
    "pos.precheck.cancel.request",
    "pos.precheck.cancel",
    "pos.payment.other",
    "pos.payment.refund",
    "pos.check.reprint",
    "pos.sync.view",
    "pos.sync.retry_failed",
]


def command_suffix():
    return uuid.uuid4().hex[:8]


def permissions_json(items):
    return json.dumps({item: True for item in sorted(set(items))}, separators=(",", ":"))


def health_check(raw_client):
    return raw_client.root_get("/health")


def create_cloud_seed(client, restaurant_name="", cashier_pin="1111", manager_pin="2222", node_device_id="", suffix=""):
    suffix = suffix or command_suffix()
    restaurant = client.post(
        "/restaurants",
        {
            "name": restaurant_name or f"Local Demo Bistro {suffix}",
            "timezone": "Europe/Moscow",
            "currency": "RUB",
            "business_day_mode": "standard",
            "business_day_boundary_local_time": "04:00",
        },
    )
    restaurant_id = restaurant["id"]
    cashier_role = client.post(
        "/roles",
        {
            "restaurant_id": restaurant_id,
            "name": f"cashier-{suffix}",
            "permissions_json": permissions_json(CASHIER_PERMISSIONS),
        },
    )
    manager_role = client.post(
        "/roles",
        {
            "restaurant_id": restaurant_id,
            "name": f"manager-{suffix}",
            "permissions_json": permissions_json(MANAGER_PERMISSIONS),
        },
    )
    cashier = client.post(
        "/employees",
        {"restaurant_id": restaurant_id, "role_id": cashier_role["id"], "name": "Demo Cashier", "pin": cashier_pin},
    )
    manager = client.post(
        "/employees",
        {"restaurant_id": restaurant_id, "role_id": manager_role["id"], "name": "Demo Manager", "pin": manager_pin},
    )
    hall = client.post("/halls", {"restaurant_id": restaurant_id, "name": "Main Hall"})
    table = client.post("/tables", {"restaurant_id": restaurant_id, "hall_id": hall["id"], "name": "T1", "seats": 2})

    catalog_tea = create_catalog_item(client, restaurant_id, "dish", f"Demo Tea {suffix}", f"DEMO-TEA-{suffix}", "portion")
    catalog_soup = create_catalog_item(client, restaurant_id, "dish", f"Demo Soup {suffix}", f"DEMO-SOUP-{suffix}", "portion")
    catalog_service = create_catalog_item(client, restaurant_id, "service", f"Demo Service {suffix}", f"DEMO-SERVICE-{suffix}", "service")

    menu_tea = create_menu_item(client, restaurant_id, catalog_tea["id"], "Demo Tea", 15000)
    menu_soup = create_menu_item(client, restaurant_id, catalog_soup["id"], "Demo Soup", 25000)
    menu_service = create_menu_item(client, restaurant_id, catalog_service["id"], "Demo Service", 5000)

    modifier_group = client.post(
        "/master-data/modifiers/groups",
        {"restaurant_id": restaurant_id, "name": "Demo Add-ons", "required": False, "min_count": 0, "max_count": 2},
    )
    modifier_option = client.post(
        "/master-data/modifiers/options",
        {
            "restaurant_id": restaurant_id,
            "modifier_group_id": modifier_group["id"],
            "name": "Lemon",
            "price_minor": 3000,
        },
    )
    modifier_binding = client.post(
        "/master-data/modifiers/bindings",
        {
            "restaurant_id": restaurant_id,
            "modifier_group_id": modifier_group["id"],
            "target_type": "menu_item",
            "target_id": menu_tea["id"],
            "sort_order": 1,
        },
    )
    publication = publish_master_data(client, restaurant_id, node_device_id, "python-seed")
    return {
        "restaurant_id": restaurant_id,
        "node_device_id": node_device_id,
        "cashier_pin": cashier_pin,
        "manager_pin": manager_pin,
        "cashier_employee_id": cashier["id"],
        "manager_employee_id": manager["id"],
        "hall_id": hall["id"],
        "table_ids": [table["id"]],
        "catalog_item_ids": [catalog_tea["id"], catalog_soup["id"], catalog_service["id"]],
        "menu_item_ids": [menu_tea["id"], menu_soup["id"], menu_service["id"]],
        "modifier_group_id": modifier_group["id"],
        "modifier_option_id": modifier_option["id"],
        "modifier_binding_id": modifier_binding["id"],
        "publication_id": publication["id"],
        "suffix": suffix,
    }


def create_catalog_item(client, restaurant_id, kind, name, sku, base_unit):
    return client.post(
        "/catalog/items",
        {"restaurant_id": restaurant_id, "type": kind, "name": name, "sku": sku, "base_unit": base_unit},
    )


def create_menu_item(client, restaurant_id, catalog_item_id, name, price):
    return client.post(
        "/menu/items",
        {
            "restaurant_id": restaurant_id,
            "catalog_item_id": catalog_item_id,
            "name": name,
            "price": price,
            "currency": "RUB",
            "availability_json": "{}",
        },
    )


def publish_master_data(client, restaurant_id, node_device_id="", published_by="python-masterdata-seed"):
    body = {"published_by": published_by}
    if node_device_id:
        body["node_device_id"] = node_device_id
    return client.post(f"/restaurants/{restaurant_id}/master-data/publish", body)


def get_edge_node_device_id(pos_client):
    status = pos_client.get("/system/provisioning-status")
    node_device_id = status.get("node_device_id", "")
    if not node_device_id:
        raise RuntimeError("POS Edge did not return node_device_id")
    return node_device_id


def provision_via_license(cloud_client, pos_client, restaurant_id, node_device_id, display_name="POS Terminal 1"):
    pairing = cloud_client.post(
        f"/restaurants/{restaurant_id}/devices/generate-pairing-code",
        {"node_device_id": node_device_id, "display_name": display_name, "expires_in_minutes": 30},
    )
    paired = pos_client.post("/system/provisioning/pair-via-license", {"pairing_code": pairing["pairing_code"]})
    return {"pairing_code": pairing["pairing_code"], "pairing_status": paired}


def provision_pos_edge(cloud_client, pos_client, cloud_base_url, restaurant_id, node_device_id, display_name="POS Terminal 1", allow_assignment_fallback=True):
    try:
        result = provision_via_license(cloud_client, pos_client, restaurant_id, node_device_id, display_name)
        result["provisioning_mode"] = "license_code"
        return result
    except Exception:
        if not allow_assignment_fallback:
            raise
    try:
        pos_client.post(
            "/system/provisioning/register-cloud",
            {"cloud_url": cloud_base_url.rstrip("/"), "display_name": display_name, "app_version": "local-python-smoke"},
            expected_status=(200,),
        )
    except Exception:
        pass
    assignment = cloud_client.post(f"/restaurants/{restaurant_id}/devices/{node_device_id}/assign", {}, expected_status=(200,))

    def check():
        status = pos_client.get("/system/provisioning-status")
        return status if status.get("paired") else None

    paired = wait_until(check, timeout_seconds=60, interval_seconds=1)
    return {"pairing_code": f"Cloud-approved:{node_device_id}", "assignment": assignment, "pairing_status": paired, "provisioning_mode": "cloud_assignment"}


def auth_headers(login, node_device_id, client_device_id):
    return {
        "X-Node-Device-ID": node_device_id,
        "X-Client-Device-ID": client_device_id,
        "X-Session-ID": login["session"]["id"],
        "X-Actor-Employee-ID": login["actor"]["employee_id"],
    }


def login_with_pin(pos_client, node_device_id, client_device_id, pin):
    return pos_client.post(
        "/auth/pin-login",
        {"node_device_id": node_device_id, "client_device_id": client_device_id, "pin": pin},
        expected_status=(201,),
    )


def verify_pos_read_model(pos_client, summary, client_device_id="python-masterdata-smoke-client"):
    login = login_with_pin(pos_client, summary["node_device_id"], client_device_id, summary["manager_pin"])
    expected_manager = summary.get("manager_employee_id")
    if expected_manager and login["actor"]["employee_id"] != expected_manager:
        raise AssertionError("manager PIN resolved unexpected employee_id")
    headers = auth_headers(login, summary["node_device_id"], client_device_id)
    halls = pos_client.get(f"/halls?restaurant_id={summary['restaurant_id']}", headers=headers)
    tables = pos_client.get(
        f"/tables?restaurant_id={summary['restaurant_id']}&hall_id={summary['hall_id']}",
        headers=headers,
    )
    menu_items = pos_client.get("/menu/items", headers=headers)
    assert_contains_id(halls, summary["hall_id"], "Cloud-created hall is not visible on POS Edge")
    for table_id in summary.get("table_ids", []):
        assert_contains_id(tables, table_id, "Cloud-created table is not visible on POS Edge")
    for menu_item_id in summary.get("menu_item_ids", []):
        assert_contains_id(menu_items, menu_item_id, "Cloud-created menu item is not visible on POS Edge")
    assert_json_contains(menu_items, summary["modifier_option_id"], "Cloud-created modifier option is not visible on POS Edge")
    return {
        "manager_employee_id": login["actor"]["employee_id"],
        "menu_items_checked": len(summary.get("menu_item_ids", [])),
        "headers": headers,
    }


def create_post_pairing_sync_item(cloud_client, summary, suffix=""):
    suffix = suffix or command_suffix()
    restaurant_id = summary["restaurant_id"]
    catalog = create_catalog_item(cloud_client, restaurant_id, "dish", f"Synced Dessert {suffix}", f"SYNC-DESSERT-{suffix}", "portion")
    menu = create_menu_item(cloud_client, restaurant_id, catalog["id"], f"Synced Dessert {suffix}", 9900)
    publication = publish_master_data(cloud_client, restaurant_id, summary.get("node_device_id", ""), "python-sync-smoke")
    return {"catalog_item_id": catalog["id"], "menu_item_id": menu["id"], "publication_id": publication["id"]}


def wait_for_menu_item(pos_client, menu_item_id, headers, timeout_seconds=90, interval_seconds=2):
    def check():
        items = pos_client.get("/menu/items", headers=headers)
        return find_by_id(items, menu_item_id)

    return wait_until(check, timeout_seconds, interval_seconds)


def verify_sync_status(pos_client, headers):
    return {
        "sync_status": pos_client.get("/sync/status", headers=headers),
        "outbox": pos_client.get("/sync/outbox?limit=5", headers=headers),
        "local_events": pos_client.get("/sync/local-events?limit=5", headers=headers),
    }


def assert_contains_id(items, expected_id, message):
    if not find_by_id(items, expected_id):
        raise AssertionError(message)


def find_by_id(value, expected_id):
    if isinstance(value, dict):
        if value.get("id") == expected_id:
            return value
        for child in value.values():
            found = find_by_id(child, expected_id)
            if found:
                return found
    if isinstance(value, list):
        for child in value:
            found = find_by_id(child, expected_id)
            if found:
                return found
    return None


def assert_json_contains(value, needle, message):
    if needle not in json.dumps(value, ensure_ascii=False):
        raise AssertionError(message)


def write_summary(path, summary):
    if not path:
        return
    with open(path, "w", encoding="utf-8") as fh:
        json.dump(summary, fh, ensure_ascii=False, indent=2)
        fh.write("\n")


def read_summary(path):
    with open(path, "r", encoding="utf-8") as fh:
        return json.load(fh)


def redacted_summary(summary):
    out = dict(summary)
    for key in ("cashier_pin", "manager_pin", "pairing_code"):
        if out.get(key):
            out[key] = "<redacted>"
    return out


def stamp_summary(summary, cloud_base_url, pos_base_url):
    out = dict(summary)
    out["cloud_base_url"] = cloud_base_url
    out["edge_base_url"] = pos_base_url
    out["generated_at_unix"] = int(time.time())
    return out
